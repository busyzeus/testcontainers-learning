# Testcontainers Go Learning Project

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

## 필요한 의존성

```bash
# 프로젝트 초기화
go mod init testcontainers-learning

# Testcontainers 관련
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/redis
go get github.com/testcontainers/testcontainers-go/modules/localstack
go get github.com/testcontainers/testcontainers-go/modules/postgres

# 데이터베이스 클라이언트
go get github.com/redis/go-redis/v9
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
go get github.com/aws/aws-sdk-go-v2/credentials
go get github.com/lib/pq
go get github.com/jmoiron/sqlx

# 테스트 유틸리티
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
```

## 구현할 기능

### Redis (redis/client.go, redis/client_test.go)
- **기본 작업**
  - Set/Get: 키-값 저장 및 조회
  - Delete: 키 삭제
  - Exists: 키 존재 확인
- **고급 기능**
  - Expire: TTL 설정
  - Increment/Decrement: 원자적 증가/감소
  - Hash 작업: HSet, HGet, HGetAll
  - List 작업: LPush, RPush, LRange

### DynamoDB Local (dynamodb/client.go, dynamodb/client_test.go)
- **테이블 관리**
  - CreateTable: 테이블 생성
  - DescribeTable: 테이블 정보 조회
  - DeleteTable: 테이블 삭제
- **항목 작업**
  - PutItem: 항목 추가
  - GetItem: 항목 조회
  - UpdateItem: 항목 업데이트
  - DeleteItem: 항목 삭제
  - Query: 조건 기반 쿼리
  - Scan: 전체 스캔

### PostgreSQL (postgres/client.go, postgres/client_test.go)
- **테이블 관리**
  - CREATE TABLE: 테이블 생성
  - DROP TABLE: 테이블 삭제
- **CRUD 작업**
  - INSERT: 데이터 삽입
  - SELECT: 데이터 조회
  - UPDATE: 데이터 업데이트
  - DELETE: 데이터 삭제
- **고급 쿼리**
  - WHERE 절을 사용한 필터링
  - JOIN 작업
  - 트랜잭션 처리

## 구현 가이드

### 1. Redis 클라이언트 구현

**redis/client.go**
```go
package redis

import (
    "context"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type Client struct {
    rdb *redis.Client
}

func NewClient(addr string) *Client {
    return &Client{
        rdb: redis.NewClient(&redis.Options{
            Addr: addr,
        }),
    }
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
    return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
    return c.rdb.Get(ctx, key).Result()
}

// ... 추가 메서드 구현
```

**redis/client_test.go**
```go
package redis

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestRedisBasicOperations(t *testing.T) {
    ctx := context.Background()
    
    // Redis 컨테이너 시작
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)
    defer redisContainer.Terminate(ctx)
    
    // 연결 정보 얻기
    endpoint, err := redisContainer.Endpoint(ctx, "")
    require.NoError(t, err)
    
    // 클라이언트 생성 및 테스트
    client := NewClient(endpoint)
    
    // Set/Get 테스트
    err = client.Set(ctx, "test-key", "test-value", 0)
    assert.NoError(t, err)
    
    value, err := client.Get(ctx, "test-key")
    assert.NoError(t, err)
    assert.Equal(t, "test-value", value)
}
```

### 2. DynamoDB 클라이언트 구현

**dynamodb/client.go**
```go
package dynamodb

import (
    "context"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Client struct {
    ddb *dynamodb.Client
}

func NewClient(cfg aws.Config) *Client {
    return &Client{
        ddb: dynamodb.NewFromConfig(cfg),
    }
}

func (c *Client) CreateTable(ctx context.Context, tableName string) error {
    _, err := c.ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
        TableName: aws.String(tableName),
        KeySchema: []types.KeySchemaElement{
            {
                AttributeName: aws.String("id"),
                KeyType:       types.KeyTypeHash,
            },
        },
        AttributeDefinitions: []types.AttributeDefinition{
            {
                AttributeName: aws.String("id"),
                AttributeType: types.ScalarAttributeTypeS,
            },
        },
        BillingMode: types.BillingModePayPerRequest,
    })
    return err
}

// ... 추가 메서드 구현
```

**dynamodb/client_test.go**
```go
package dynamodb

import (
    "context"
    "testing"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/localstack"
)

func TestDynamoDBBasicOperations(t *testing.T) {
    ctx := context.Background()
    
    // LocalStack 컨테이너 시작 (DynamoDB 포함)
    localstackContainer, err := localstack.RunContainer(ctx,
        testcontainers.WithImage("localstack/localstack:3.0"),
    )
    require.NoError(t, err)
    defer localstackContainer.Terminate(ctx)
    
    // AWS 설정
    provider, err := testcontainers.NewDockerProvider()
    require.NoError(t, err)
    defer provider.Close()

    host, err := provider.DaemonHost(ctx)
    require.NoError(t, err)
    
    mappedPort, err := localstackContainer.MappedPort(ctx, "4566/tcp")
    require.NoError(t, err)
    
    endpoint := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
    
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
    )
    require.NoError(t, err)
    
    // 클라이언트 생성 및 테스트
    client := NewClient(cfg)
    
    // 테이블 생성 테스트
    err = client.CreateTable(ctx, "test-table")
    assert.NoError(t, err)
}
```

### 3. PostgreSQL 클라이언트 구현

**postgres/client.go**
```go
package postgres

import (
    "context"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

type Client struct {
    db *sqlx.DB
}

func NewClient(connStr string) (*Client, error) {
    db, err := sqlx.Connect("postgres", connStr)
    if err != nil {
        return nil, err
    }
    
    return &Client{db: db}, nil
}

func (c *Client) CreateTable(ctx context.Context, tableName string) error {
    query := `
        CREATE TABLE IF NOT EXISTS ` + tableName + ` (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `
    _, err := c.db.ExecContext(ctx, query)
    return err
}

// ... 추가 메서드 구현
```

**postgres/client_test.go**
```go
package postgres

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgreSQLBasicOperations(t *testing.T) {
    ctx := context.Background()
    
    // PostgreSQL 컨테이너 시작
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(5*time.Second),
        ),
    )
    require.NoError(t, err)
    defer postgresContainer.Terminate(ctx)
    
    // 연결 문자열 얻기
    connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    
    // 클라이언트 생성 및 테스트
    client, err := NewClient(connStr)
    require.NoError(t, err)
    
    // 테이블 생성 테스트
    err = client.CreateTable(ctx, "users")
    assert.NoError(t, err)
}
```

## 실행 방법

```bash
# 모든 테스트 실행
go test ./...

# 특정 패키지 테스트
go test ./redis -v
go test ./dynamodb -v
go test ./postgres -v

# 통합 테스트 실행
go test ./examples -v

# 상세 로그와 함께 실행
go test ./... -v -count=1
```

## 학습 포인트

### Testcontainers 핵심 개념
1. **컨테이너 생명주기**: RunContainer, Terminate
2. **대기 전략**: wait.ForLog, wait.ForHTTP
3. **포트 매핑**: MappedPort, Endpoint
4. **환경 변수 설정**: WithEnv
5. **초기화 스크립트**: WithInitScript

### 베스트 프랙티스
- `defer container.Terminate(ctx)`로 리소스 정리 보장
- `require`와 `assert`를 적절히 구분하여 사용
- 테스트 격리를 위해 각 테스트마다 새 컨테이너 사용 또는 데이터 정리
- Context timeout 설정으로 무한 대기 방지
- CI/CD 환경을 고려한 대기 전략 구성

### 확장 아이디어
- 여러 컨테이너를 동시에 사용하는 통합 테스트
- 네트워크로 연결된 컨테이너 간 통신 테스트
- 커스텀 Docker 이미지 사용
- 볼륨 마운트를 통한 데이터 영속성 테스트
- 병렬 테스트 실행 최적화

## 참고 자료

- [Testcontainers Go 공식 문서](https://golang.testcontainers.org/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [sqlx 문서](https://jmoiron.github.io/sqlx/)

## 트러블슈팅

### Docker 관련
- Docker Desktop이 실행 중인지 확인
- Docker 소켓 권한 확인 (Linux)
- WSL2에서 Docker 연동 확인 (Windows)

### 테스트 관련
- 포트 충돌: 이미 사용 중인 포트가 있는지 확인
- 타임아웃: 대기 전략 조정 또는 timeout 증가
- 메모리 부족: Docker Desktop의 리소스 할당 확인

### 네트워크 관련
- LocalStack 접근 시 endpoint URL 정확히 설정
- 컨테이너 간 통신 시 Docker 네트워크 구성 확인
