# Testcontainers Go 학습 프로젝트

Go와 Testcontainers를 사용하여 Redis, DynamoDB Local, PostgreSQL의 기본 기능을 테스트하는 학습 프로젝트입니다.

## 프로젝트 개요

이 프로젝트는 Testcontainers를 활용하여 세 가지 주요 데이터 저장소의 기본 CRUD 작업을 테스트합니다:
- **Redis**: 키-값 저장소 및 캐싱
- **DynamoDB Local**: NoSQL 문서 데이터베이스
- **PostgreSQL**: 관계형 데이터베이스

## 디렉토리 구조

```
testcontainers-learning/
├── go.mod
├── go.sum
├── README.md
├── CLAUDE.md
├── redis/
│   ├── client.go          # Redis 클라이언트 래퍼
│   └── client_test.go     # Redis 테스트
├── dynamodb/
│   ├── client.go          # DynamoDB 클라이언트 래퍼
│   └── client_test.go     # DynamoDB 테스트
├── postgres/
│   ├── client.go          # PostgreSQL 클라이언트 래퍼
│   └── client_test.go     # PostgreSQL 테스트
└── examples/
    └── integration_test.go # 통합 테스트 예제
```

## 전제 조건

- Go 1.21 이상
- Docker Desktop (또는 Docker Engine)
- 충분한 메모리 (최소 4GB 권장)

## 설치

```bash
# 프로젝트 클론
git clone <repository-url>
cd testcontainers-learning

# 의존성 다운로드
go mod download
```

## 실행 방법

### 전체 테스트 실행

```bash
# 모든 테스트 실행
go test ./...

# 상세 로그와 함께 실행
go test ./... -v

# 캐시 무시하고 실행
go test ./... -v -count=1
```

### 개별 패키지 테스트

```bash
# Redis 테스트만 실행
go test ./redis -v

# DynamoDB 테스트만 실행
go test ./dynamodb -v

# PostgreSQL 테스트만 실행
go test ./postgres -v

# 통합 테스트만 실행
go test ./examples -v
```

### 특정 테스트 실행

```bash
# Redis 기본 작업 테스트
go test ./redis -v -run TestRedisBasicOperations

# DynamoDB PutItem/GetItem 테스트
go test ./dynamodb -v -run TestDynamoDBPutAndGetItem

# PostgreSQL 트랜잭션 테스트
go test ./postgres -v -run TestPostgreSQLTransaction

# 캐시 어사이드 패턴 테스트
go test ./examples -v -run TestCacheAsidePattern
```

## 구현된 기능

### Redis (redis/client.go)
- **기본 작업**
  - `Set/Get`: 키-값 저장 및 조회
  - `Delete`: 키 삭제
  - `Exists`: 키 존재 확인
- **고급 기능**
  - `Expire`: TTL 설정
  - `Increment/Decrement`: 원자적 증가/감소
  - `HSet/HGet/HGetAll`: Hash 작업
  - `LPush/RPush/LRange`: List 작업

### DynamoDB (dynamodb/client.go)
- **테이블 관리**
  - `CreateTable`: 테이블 생성
  - `DescribeTable`: 테이블 정보 조회
  - `DeleteTable`: 테이블 삭제
- **항목 작업**
  - `PutItem`: 항목 추가
  - `GetItem`: 항목 조회
  - `UpdateItem`: 항목 업데이트
  - `DeleteItem`: 항목 삭제
  - `Query`: 조건 기반 쿼리
  - `Scan`: 전체 스캔

### PostgreSQL (postgres/client.go)
- **테이블 관리**
  - `CreateTable`: 테이블 생성
  - `DropTable`: 테이블 삭제
- **CRUD 작업**
  - `InsertUser`: 데이터 삽입
  - `GetUser/GetAllUsers`: 데이터 조회
  - `UpdateUser`: 데이터 업데이트
  - `DeleteUser`: 데이터 삭제
- **고급 기능**
  - `GetUsersByNamePattern`: WHERE 절을 사용한 필터링
  - `ExecuteInTransaction`: 트랜잭션 처리

### 통합 테스트 (examples/integration_test.go)
- **다중 컨테이너 통합 테스트**: Redis, PostgreSQL, DynamoDB를 모두 사용하는 사용자 등록 및 세션 관리 시나리오
- **캐시 어사이드 패턴**: Redis를 캐시로 사용하고 PostgreSQL을 주 데이터 저장소로 사용
- **분산 카운터 패턴**: Redis를 이용한 원자적 카운터 구현

## 테스트 예시

### Redis 테스트
```go
// 기본 Set/Get 테스트
err = client.Set(ctx, "test-key", "test-value", 0)
assert.NoError(t, err)

value, err := client.Get(ctx, "test-key")
assert.NoError(t, err)
assert.Equal(t, "test-value", value)
```

### DynamoDB 테스트
```go
// 항목 추가 테스트
item := map[string]types.AttributeValue{
    "id":    &types.AttributeValueMemberS{Value: "user-1"},
    "name":  &types.AttributeValueMemberS{Value: "John Doe"},
}
err = client.PutItem(ctx, tableName, item)
assert.NoError(t, err)
```

### PostgreSQL 테스트
```go
// 사용자 추가 및 조회 테스트
id, err := client.InsertUser(ctx, "users", "John Doe", "john@example.com")
require.NoError(t, err)

user, err := client.GetUser(ctx, "users", id)
assert.NoError(t, err)
assert.Equal(t, "John Doe", user.Name)
```

## 학습 포인트

### Testcontainers 핵심 개념
1. **컨테이너 생명주기**: `Run`, `TerminateContainer`
2. **대기 전략**: `wait.ForLog`, `wait.ForHTTP`
3. **포트 매핑**: `MappedPort`, `Endpoint`
4. **환경 변수 설정**: `WithDatabase`, `WithUsername`, `WithPassword`

### 베스트 프랙티스
- `defer` 문으로 리소스 정리 보장
- `require`와 `assert`를 적절히 구분하여 사용
- 테스트 격리를 위해 각 테스트마다 새 컨테이너 사용
- Context timeout 설정으로 무한 대기 방지
- CI/CD 환경을 고려한 대기 전략 구성

## 트러블슈팅

### Docker 관련
- Docker Desktop이 실행 중인지 확인
- Docker 소켓 권한 확인 (Linux)
- WSL2에서 Docker 연동 확인 (Windows)
- /tmp에서 실행이 되는 경우 mkdir -p ~/tmp && export TMPDIR=~/tmp 실행하고 테스트 

### 테스트 관련
- **포트 충돌**: 이미 사용 중인 포트가 있는지 확인
- **타임아웃**: 대기 전략 조정 또는 timeout 증가
- **메모리 부족**: Docker Desktop의 리소스 할당 확인

### 네트워크 관련
- LocalStack 접근 시 endpoint URL 정확히 설정
- 컨테이너 간 통신 시 Docker 네트워크 구성 확인

## 참고 자료

- [Testcontainers Go 공식 문서](https://golang.testcontainers.org/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [sqlx 문서](https://jmoiron.github.io/sqlx/)

## 라이선스

MIT License
