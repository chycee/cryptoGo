# CryptoGo 코드 리뷰 보고서

> 79개 Go 소스 파일 전체를 체계적으로 검토한 결과입니다.

---

## 🔴 Critical — 즉시 수정 필요

### 1. Execution 인터페이스 중복 및 불일치

두 개의 서로 다른 `Execution` 인터페이스가 존재하며, 시그니처가 다릅니다.

| 위치 | 메서드 시그니처 |
|------|----------------|
| [execution.go](file:///home/choyc/workspace/cryptoGo/internal/domain/execution.go) | `ExecuteOrder(ctx, Order)`, `CancelOrder(ctx, orderID, symbol)`, `Close()` |
| [interface.go](file:///home/choyc/workspace/cryptoGo/internal/execution/interface.go) | `SubmitOrder(ctx, Order)`, `CancelOrder(ctx, orderID)` — `symbol` 없음, `Close()` 없음 |

- `PaperExecution`, `RealExecution`은 `domain.Execution`을 구현
- `MockExecution`은 `execution.Execution`을 구현 (`SubmitOrder`, `Close()` 없음)
- **둘 중 하나를 삭제하고 통일해야 합니다.** 현재 `MockExecution`은 `domain.Execution` 타입에 할당 불가합니다.

---

### 2. Sequencer의 Race Condition (Data Race 가능)

[sequencer.go](file:///home/choyc/workspace/cryptoGo/internal/engine/sequencer.go)에서:

```go
// Run()은 s.markets, s.nextSeq를 뮤텍스 없이 수정
func (s *Sequencer) processEvent(ev event.Event) {
    s.markets[e.Symbol] = state  // 뮤텍스 없음
    s.nextSeq++                   // 뮤텍스 없음
}

// 외부 읽기는 RWMutex를 사용
func (s *Sequencer) GetMarketState(symbol string) { s.mu.RLock() ... }
func (s *Sequencer) GetNextSeq() uint64 { s.mu.RLock() ... }
```

`Run()`은 단일 goroutine이지만, `GetMarketState()`/`GetNextSeq()`는 다른 goroutine에서 호출됩니다. **Writer가 Lock을 잡지 않으므로 Reader의 RLock은 보호 효과가 없습니다.** Go의 race detector가 이를 감지할 것입니다.

> [!CAUTION]
> `processEvent`에서 `s.mu.Lock()`을 사용하거나, `atomic` 연산으로 전환해야 합니다.

---

### 3. Sequence Number Race Condition (여러 Worker 간)

[main.go](file:///home/choyc/workspace/cryptoGo/cmd/app/main.go#L64):

```go
nextSeq := uint64(1)
// 모든 worker가 &nextSeq를 공유
exchangeRateClient := infra.NewExchangeRateClient(seq.Inbox(), &nextSeq)
upbitWorker := upbit.NewWorker(cfg.API.Upbit.Symbols, seq.Inbox(), &nextSeq)
bitgetSpotWorker := bitget.NewSpotWorker(cfg.API.Bitget.Symbols, seq.Inbox(), &nextSeq)
bitgetFuturesWorker := bitget.NewFuturesWorker(cfg.API.Bitget.Symbols, seq.Inbox(), &nextSeq)
```

`quant.NextSeq`는 `atomic.AddUint64`를 사용하므로 원자적 **증가**는 안전하지만, **Sequencer의 `ValidateSequence`**는 연속 증가를 기대합니다. 4개 Worker가 동시에 seq를 발급하면 Sequencer에 도착하는 순서가 비결정적이므로 sequence gap이 빈번하게 발생할 수 있습니다.

---

### 4. Event Pool Use-After-Release 위험

[spot_worker.go](file:///home/choyc/workspace/cryptoGo/internal/infra/bitget/spot_worker.go#L77-L89) 등에서:

```go
ev := event.AcquireMarketUpdateEvent()
ev.Symbol = symbol
// ...
select {
case w.inbox <- ev:     // Sequencer가 소유권을 가짐
default:
    event.ReleaseMarketUpdateEvent(ev)  // 드롭 시 풀에 반환
}
```

**문제**: 성공적으로 `inbox`에 전송된 이벤트는 **누가 `Release`하는지 불분명**합니다. `Sequencer.processEvent()`에서 Release를 호출하지 않으므로, **이벤트가 풀에 영원히 반환되지 않습니다** (메모리 누수). WAL에 저장된 후에 Release해야 합니다.

---

## 🟠 High — 중요 이슈

### 5. PaperExecution의 하드코딩된 심볼 파싱

[paper.go](file:///home/choyc/workspace/cryptoGo/internal/execution/paper.go#L87-L88):

```go
baseSymbol := order.Symbol[:3]  // "BTC" from "BTCUSDT"
quoteSymbol := order.Symbol[3:] // "USDT" from "BTCUSDT"
```

- `DOGE`, `SHIB` 등 4글자 이상 심볼에서 잘못 파싱됨 (예: `DOGEUSDT` → `DOG` + `EUSDT`)
- 별도의 심볼 파서 또는 구분자(`-`) 기반 파싱이 필요

---

### 6. Bitget API 서명에서 Query String 처리 미흡

[signer.go](file:///home/choyc/workspace/cryptoGo/internal/infra/bitget/signer.go#L46-L64)의 `GenerateHeaders`:

```go
payload := timestamp + method + path + query + body
```

[client.go](file:///home/choyc/workspace/cryptoGo/internal/infra/bitget/client.go#L254)의 `doRequest`:

```go
headers := c.signer.GenerateHeaders(method, path, "", bodyStr) // query = "" 항상 빈 문자열
```

`GetBalance`에서 `path = "/api/v2/mix/account/accounts?productType=USDT-FUTURES"` 처럼 query가 path에 포함되어 있으므로, 실제로는 동작하지만:
- Bitget API 문서의 서명 규칙(`timestamp + method + requestPath + "?" + queryString + body`)과 다름
- GET 요청에 query 파라미터가 별도로 있을 때 서명 불일치로 인증 실패 가능

---

### 7. Lock File이 크래시 시 정리되지 않음

[paths.go](file:///home/choyc/workspace/cryptoGo/internal/infra/paths.go#L62-L86):

```go
unlock, err := infra.CreateLockFile(workDir)
_ = unlock  // closer를 무시!
```

- `O_EXCL` 플래그로 생성하므로 프로세스 크래시 후 재시작 불가 (수동 삭제 필요)
- `flock` 시스템콜을 사용하면 프로세스 종료 시 자동 해제됨

---

### 8. `LoadEvents`가 `MarketUpdateEvent`만 반환

[store.go](file:///home/choyc/workspace/cryptoGo/internal/storage/store.go#L122):

```go
func (s *EventStore) LoadEvents(ctx context.Context, fromSeq uint64) ([]*event.MarketUpdateEvent, error) {
```

- `OrderUpdateEvent`, `BalanceUpdateEvent` 등은 무시됨
- `RecoverFromWAL`에서 `LoadEvents`를 사용하므로, 주문/잔고 이벤트가 복구되지 않음
- 반환 타입을 `[]event.Event`로 변경해야 함

---

### 9. Replayer가 DB를 두 번 열고 닫지 않음

[replayer.go](file:///home/choyc/workspace/cryptoGo/backtest/replayer.go#L21-L35):

```go
db, err := sql.Open("sqlite", dbPath)        // 첫 번째
store, err := storage.NewEventStore(dbPath)   // 두 번째 (같은 DB)
```

- `db`와 `store`가 같은 DB 파일을 별도로 열어서 리소스 낭비
- `store` 필드는 사용되지 않음
- `db`와 `store` 모두 `Close()` 미호출

---

## 🟡 Medium — 개선 권장

### 10. `CalculateTotalEquity` 의 Overflow 가능성

[balance.go](file:///home/choyc/workspace/cryptoGo/internal/domain/balance.go#L135):

```go
assetValue := safe.SafeMul(balance.AmountSats, price)
```

`AmountSats` (10^8 단위) × `price` (10^6 단위) = 10^14 단위. BTC 가격이 1억 KRW이면 `100_000_000 * 100_000_000_000_000 = 10^22` → **int64 overflow** (Max: ~9.2 × 10^18). `SafeMul`이 panic하므로 프로그램 크래시.

---

### 11. `parseFixedPoint` 에러 무시 (`quant_types.go`)

[quant_types.go](file:///home/choyc/workspace/cryptoGo/pkg/quant/quant_types.go#L92):

```go
intPart, _ := strconv.ParseInt(parts[0], 10, 64)  // 에러 무시
fracPart, _ := strconv.ParseInt(fracStr, 10, 64)   // 에러 무시
```

잘못된 문자열 입력 시 0이 반환되어 가격이 0으로 설정됨 → 잘못된 매매 신호 발생 가능

---

### 12. `ExchangeRateClient`가 Config를 사용하지 않음

[main.go](file:///home/choyc/workspace/cryptoGo/cmd/app/main.go#L67):

```go
exchangeRateClient := infra.NewExchangeRateClient(seq.Inbox(), &nextSeq)
```

Config에 `exchange_rate` URL과 `poll_interval_sec`이 정의되어 있지만, `NewExchangeRateClient`는 하드코딩된 기본값을 사용합니다. `NewExchangeRateClientWithConfig`를 대신 사용해야 합니다.

---

### 13. `PaperExecution`이 `domain.Execution`과 완전히 호환되지 않음

- `NewPaperExecution(initialBalance)`에서 `initialBalance`를 `quant.ToPriceMicros(100_000_000.0)`로 계산
- 이는 KRW 1억 → Micros로 변환한 값 (`100_000_000_000_000`)
- 하지만 Balance에 `Credit(int64(initialBalance), 0)`로 넣는데, 이 값은 "Sats" 필드에 들어감 → **단위 불일치**

---

### 14. `json.Marshal` 에러 무시

[spot_worker.go](file:///home/choyc/workspace/cryptoGo/internal/infra/bitget/spot_worker.go#L51), [futures_worker.go](file:///home/choyc/workspace/cryptoGo/internal/infra/bitget/futures_worker.go#L52), [upbit_worker.go](file:///home/choyc/workspace/cryptoGo/internal/infra/upbit/upbit_worker.go#L79):

```go
b, _ := json.Marshal(req)  // 에러 무시
```

---

### 15. `IsConnected()` 인터페이스 메서드 미구현

[interfaces.go](file:///home/choyc/workspace/cryptoGo/internal/domain/interfaces.go) `ExchangeWorker` 인터페이스에 `IsConnected() bool`이 있지만, 어떤 Worker도 이를 구현하지 않음. 실제로 이 인터페이스를 사용하는 곳도 없어HistoryWorkers는 `domain.ExchangeWorker`로 타입 체크 불가.

---

## 🔵 Low — 코드 품질 개선

### 16. `AlertConfig.active` 필드가 JSON으로 직렬화되지 않음

[alert.go](file:///home/choyc/workspace/cryptoGo/internal/domain/alert.go#L12): `active bool` (소문자) → JSON 비공개. persistente alert 복원 시 항상 `false`로 시작.

### 17. `app_config.go`의 Windows 줄바꿈 (CRLF)

일부 파일이 CRLF (`\r\n`)를 사용, 나머지는 LF — .gitattributes 또는 editorconfig로 통일 권장.

### 18. `DumpState`에서 `VerifyAll()` 호출 시 Panic 가능

[sequencer.go](file:///home/choyc/workspace/cryptoGo/internal/engine/sequencer.go#L245): Panic 핸들러 안에서 `DumpState`를 호출하고, `DumpState` 안에서 `VerifyAll()`이 또 panic하면 **이중 panic**으로 덤프가 실패합니다.

### 19. `PlaceOrder` 가격 포매팅에서 음수 처리 미흡

[client.go](file:///home/choyc/workspace/cryptoGo/internal/infra/bitget/client.go#L87):

```go
priceStr := fmt.Sprintf("%d.%06d", order.PriceMicros/1_000_000, order.PriceMicros%1_000_000)
```

음수 가격에서 `%06d`가 음수를 처리하지 못함 (예: `-1234567` → `"-1.234567"` 대신 `"-2.765433"` 등 잘못된 결과).

### 20. `WeakSecretConfig` 주석 중복

[config_secret.go](file:///home/choyc/workspace/cryptoGo/internal/infra/config_secret.go#L10-L11): 동일 주석 2줄 연속.

---

## 📋 요약

| 심각도 | 개수 | 대표 이슈 |
|--------|------|-----------|
| 🔴 Critical | 4 | Interface 중복, Race Condition, Event 메모리 누수 |
| 🟠 High | 5 | 심볼 파싱 오류, API 서명, Lock File, WAL 복구 불완전 |
| 🟡 Medium | 6 | Overflow, 에러 무시, 단위 불일치 |
| 🔵 Low | 5 | JSON 직렬화, CRLF, 이중 Panic, 주석 중복 |

> **우선순위 권장**: Critical #1 → #2 → #3 → #4 → High #5 순으로 수정
