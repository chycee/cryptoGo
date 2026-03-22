# CryptoGo 다음 작업 목록

> 2026-03-18 코드 리뷰 후 보류/개선 항목 정리

---

## 1. 보류된 수정 (2건)

### #13 PaperExecution 단위 불일치

[paper.go](file:///home/choyc/workspace/cryptoGo/internal/execution/paper.go) `NewPaperExecution`:

```go
initialBalance := quant.ToPriceMicros(100_000_000.0)  // KRW 1억 → PriceMicros
p.balances.Get("KRW").Credit(int64(initialBalance), 0) // Sats 필드에 넣음
```

- `PriceMicros`(10^6 스케일) 값을 `AmountSats`(10^8 스케일) 필드에 넣고 있음
- **해결 방안**: `initialBalance`를 Config에서 받거나, 명시적으로 Sats 단위로 변환 후 저장
- **영향 범위**: PaperExecution 내부만 (실제 거래 미영향)

### ~~#17 CRLF/LF 줄바꿈 혼용~~ ✅ 완료 (2026-03-22)

- `.gitattributes` 추가 (`* text=auto eol=lf`)
- 72개 파일 CRLF → LF 일괄 변환 완료
- 전체 테스트 PASS 확인

---

## 2. 구조 개선 (4건)

### A. 테스트 커버리지 강화

현재 테스트가 부족한 핵심 영역:

| 대상 | 현재 | 필요한 테스트 |
|------|------|--------------|
| Sequencer 핫패스 | 기본 이벤트 처리만 | 다중 심볼, 전략 연동, WAL 복구 통합 |
| Strategy | SMA 계산만 | 시그널 생성 → 주문 흐름 E2E |
| PaperExecution | 단일 BUY/SELL | 연속 매매, 잔고 부족, 다중 심볼 |
| CircuitBreaker | 상태 전이 | Half-Open 복구, 동시성 |
| WebSocket Worker | 없음 | 재연결, 메시지 드롭, ping timeout |

**우선순위**: Sequencer 통합 테스트 → Strategy E2E → Worker 재연결

### B. Graceful Shutdown 강화

[main.go](file:///home/choyc/workspace/cryptoGo/cmd/app/main.go)에서 현재:

```go
defer upbitWorker.Disconnect()  // 즉시 끊김
```

**필요한 것**:
1. Sequencer inbox를 drain (진행 중 이벤트 WAL 기록 보장)
2. Worker 종료 순서 보장 (Worker 먼저 → Sequencer → Store)
3. Shutdown timeout 설정 (무한 대기 방지)

```go
// 개선안 예시
func gracefulShutdown(cancel context.CancelFunc, seq *Sequencer, workers []ExchangeWorker) {
    cancel()                    // 1. 새 이벤트 수신 중단
    seq.DrainAndClose()         // 2. 남은 이벤트 처리 + WAL 기록
    for _, w := range workers { // 3. Worker 종료
        w.Disconnect()
    }
}
```

### C. 설정 검증 (Config Validation)

현재 Config 로딩 후 필수 필드 검증이 없음. 잘못된 설정이 런타임까지 전파됨.

**추가할 검증**:
- `symbols` 비어있으면 Fail Fast
- `trading.mode`가 `paper`/`demo`/`real` 이외이면 에러
- `demo`/`real` 모드에서 API key 없으면 시작 차단
- Symbol 형식 검증 (`BASE-QUOTE` 패턴)

### D. 관측성 (Observability) 확장

현재 `Metrics` struct는 있지만 외부 노출이 없음 (Pprof만 있음).

**추가할 것**:
- `/metrics` HTTP 엔드포인트 (JSON 형태 Snapshot 노출)
- 주기적 메트릭 로깅 (30초마다 이벤트 처리량, 지연시간 등)
- 에러율 임계치 알림 (로그 기반)

---

## 우선순위 권장

```
1순위: #13 단위 불일치  (30분, 버그)
2순위: B. Graceful Shutdown  (1시간, 안정성)
3순위: C. 설정 검증  (1시간, 안정성)
4순위: A. 테스트 강화  (2~3시간, 품질)
5순위: D. 관측성  (1시간, 운영)
6순위: ~~#17 CRLF  (5분, 코드 정리)~~ ✅
```
