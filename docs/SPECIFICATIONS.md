# 크립토 고 (Crypto Go) 기술 명세 및 운영 가이드

---

## 1. 도메인 및 API 명세 (Domain & API)

### 1.1 핵심 데이터 모델 (Internal Entities)

#### `Ticker` (ticker.go)
거래소별 가공된 시세 데이터 객체입니다.
- `Symbol`: 통합 심볼 (예: "BTC")
- `Price`: 현재가 (`decimal.Decimal`)
- `Volume`: 24시간 거래량
- `Exchange`: 거래소 구분 ("UPBIT", "BITGET_S", "BITGET_F")
- `Precision`: 가격 표시 소수점 자릿수
- `FundingRate`: 펀딩비 (선물 전용)

#### `MarketData` (ticker.go)
단일 종목의 통합 정보를 담은 엔티티입니다.
- `Symbol`: 통합 심볼
- `Upbit`, `BitgetS`, `BitgetF`: 각 거래소별 `Ticker` 정보
- `Premium`: 김치 프리미엄 (%)
- `IsFavorite`: 즐겨찾기 여부

### 1.2 서비스 인터페이스 (Service Interface)

#### `PriceService` (price_service.go)
- `GetTickerChan()`: 비동기 시세 입력을 위한 채널 반환
- `ProcessTickers(tickers)`: 시세 데이터를 처리하고 프리미엄을 계산 (비동기 처리기에서 호출)
- `GetAllData()`: 현재 메모리에 유지된 모든 마켓 데이터 반환
- `UpdateExchangeRate(rate)`: 실시간 환율 반영 및 모든 데이터 재계산

---

## 2. 테스트 전략 (Testing Strategy)

### 🧪 단위 테스트 (Unit Tests)
- **에지 케이스**: 환율 0, 음수 가격, 가격 정밀도 충돌 시의 산식 안정성 검증.
- **정밀도**: `Decimal` 연산 시 부동 소수점 오차 발생 여부 상시 체크.

### 🔗 통합 테스트 (Integration)
- **WebSocket Mocking**: `httptest`를 이용해 실제 서버 없이 재연결 및 파싱 로직 검증.
- **레이스 컨디션**: `go test -race`를 통한 동시성 안전 보장.

---

## 3. 모니터링 및 관측성 (Observability)

### 📊 로깅 정책 (Logging)
- **Structured Log**: `slog`를 사용하며 모든 로그에 `session_id` 컨텍스트 포함.
- **로그 레벨**:
    - `DEBUG`: 원시 메시지 (Trace용)
    - `WARN`: 재연결, 채널 버퍼 임계치 도달, 심볼 제한
    - `ERROR`: 인증 실패, 네트워크 영구 장애

### 📈 메트릭 (Metrics)
- **주요 지표**: 수신 티커 수, 처리 지연 시간(Latency), 재연결 시도 횟수.
- **상태 확인**: Liveness 및 Readiness 프로브 설계 반영.

---

## 4. 시스템 고도화 로드맵 (Future Roadmap)

- **Graceful Shutdown**: 종료 시 리소스 해제 및 대기 데이터 완전 처리 보장.
- **에러 정책**: `Transient`, `Critical`, `Data` 타입별 차등화된 복구 전략.
- **데이터 영속성**: 앱 재시작 시 히스토리 유지를 위한 스냅샷 저장 기능.

---
*Last Updated: 2026-01-06*
