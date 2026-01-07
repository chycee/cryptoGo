# 🚀 CryptoGo: Indie Quant Framework

**CryptoGo**는 초고속 의사결정과 완벽한 검증(Backtest is Reality)을 목표로 하는 **Go 언어 기반의 결정론적(Deterministic) 퀀트 트레이딩 프레임워크**입니다.

"복잡함은 적이다(Complexity is the Enemy)"라는 철학 아래, **단일 스레드 시퀀서(Single-Threaded Sequencer)** 아키텍처를 채택하여 동시성 문제를 원천 차단했습니다.

---

## 🏛️ 아키텍처 (Architecture)

모든 데이터 흐름은 **Sequencer**라고 불리는 단일 파이프라인(Hotpath)을 통과합니다.

```mermaid
graph LR
    subgraph Inputs [I/O Layer]
        UB[Upbit WS] -->|Chan| Inbox
        BG[Bitget WS] -->|Chan| Inbox
    end

    subgraph Core [Sequencer (Single Thread)]
        Inbox((Inbox)) -->|Event| Check[Gap Check]
        Check -->|Event| WAL[(SQLite WAL)]
        WAL --> Logic{Process Event}
        Logic -->|MarketState| Strategy[Strategy Mode]
        Strategy -->|Action| Exec[Execution]
        Logic -->|Update| State[In-Memory State]
    end

    subgraph Output [Actions]
        Exec -->|Order| API[Exchange API]
        State -->|Snapshot| UI[User Interface]
    end
```

### 핵심 컴포넌트
1.  **Sequencer (엔진)**: 모든 이벤트(시세, 주문체결 등)를 **단 하나의 고루틴**에서 순차적으로 처리합니다. Mutex가 전혀 필요 없습니다.
2.  **Strategies (두뇌)**: 시세 변동 시 동기적(Synchronous)으로 호출되며, 매수/매도 신호(Action)를 반환합니다.
3.  **Infrastructure (손발)**: 외부 거래소 통신 모듈입니다. 거래소별(Provider)로 독립적인 패키지로 격리되어 있습니다.

---

## 🛠️ 모듈별 상세 (Modules)

이 프레임워크는 **안전성(Safety)**과 **성능(Performance)**을 최우선으로 설계되었습니다.

### 1. `pkg/safe` & `pkg/quant` (Core Data)
*   **Integer Only**: 모든 돈과 수량은 `int64`입니다. 부동소수점(`float`) 사용은 **엄격히 금지**됩니다.
    *   `PriceMicros`: 1 KRW = 1,000,000 (마이크로 단위)
    *   `QtySats`: 1 BTC = 100,000,000 (사토시 단위)
*   **Safety Math**: `SafeAdd`, `SafeMul` 등의 함수는 오버플로우 발생 시 **즉시 패닉(Panic)**을 일으켜 시스템을 멈춥니다.

### 2. `internal/engine` (Sequencer)
*   **Hotpath**: 주문 처리의 핵심 경로입니다.
*   **Gap Detection**: 수신된 이벤트의 시퀀스 번호(Seq)가 비어있다면, 데이터 유실로 간주하고 즉시 셧다운합니다.
*   **Zero-Alloc Policy**: 런타임 중 힙 메모리 할당(`GC Overhead`)을 최소화합니다.

### 3. `internal/infra` (Gateways)
거래소별로 프로토콜이 다르므로, 자산군(Asset Class)이 아닌 **제공자(Provider)** 기준으로 패키지를 분리했습니다.

*   `internal/infra/upbit`: 업비트 웹소켓
*   `internal/infra/bitget`: 비트겟 웹소켓 (Spot + Futures)
*   `internal/infra/ls`: LS증권 (Webhook/API Stub - Future Impl)

### 4. `internal/strategy` (Trading Logic)
*   **Interface**: 모든 전략은 `OnMarketUpdate(state)` 메서드를 구현해야 합니다.
*   **Zero-Alloc Pattern**: `SMACrossStrategy` 레퍼런스 구현체는 **Ring Buffer**를 사용하여, 루프 내에서 메모리 할당 없이 이동평균을 계산합니다.

---

## 🚀 시작하기 (Getting Started)

### 요구 사항
*   Go 1.21 이상
*   Windows / Linux / macOS

### 실행 방법
```bash
# 1. 의존성 설치
go mod tidy

# 2. 실행 (기본 SMA 전략 탑재됨)
go run cmd/app/main.go
# -> 로그에서 "Sequencer started" 및 "STRATEGY_ACTION" 확인 가능
```

### 테스트 실행
```bash
# 전체 유닛 테스트 (Race Detector 포함 권장)
go test -v -race ./...
```

---

*Created by Indie Quant Team based on Deterministic Architecture.*
