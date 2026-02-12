CLAUDE.md

# CLAUDE.md

## 0. 문서 목적

이 문서는 **Claude Code + everything-claude-code 플러그인**이 이 레포지토리를 자동 구현/수정할 때 따라야 하는 **개발 규칙(행동 지침서)** 입니다.

- README.md는 프로젝트 소개/실행 방법을 담습니다.
- CLAUDE.md는 AI Agent가 **추측하지 않고 일관된 방식으로 구현**하도록 강제하는 규칙을 담습니다.

---

## 1. 프로젝트 개요

이 레포지토리는 Todo(할 일) 관리 앱의 백엔드 API 프로젝트입니다.

주요 기능:

- 유저 가입/로그인 (Cognito 기반: 이메일 + 소셜 로그인)
- 유저 프로필 관리
- Todo CRUD
- Todo 상태 변경 (완료/미완료)
- due time 기반 알림 (이메일/푸시)

환경:

- local
- alpha (dev)
- beta (staging)
- prod (release)

---

## 2. 목표 아키텍처

- API 서버: Go (ECS Fargate) + ALB
- DB: Aurora PostgreSQL Serverless v2
- 인증(Auth): AWS Cognito User Pool
- 알림 스케줄링: EventBridge Scheduler → Lambda Worker
- 이메일 발송: SES
- 푸시 발송: SNS
- 스토리지: S3
- CDN: CloudFront
- 로깅/관측: CloudWatch Logs/Metrics/Alarms
- IaC: Terraform
- CI/CD: GitHub Actions

---

## 3. 프로젝트 핵심 원칙

### 3.1 Stateless 원칙

API 서버는 Stateless로 구현해야 합니다.

- 서버 세션(Session) 저장 금지
- 서버 로컬 파일 저장 금지
- 유저 식별은 Cognito JWT 기반

---

### 3.2 No Guessing Policy (추측 금지)

요구사항이 명확하지 않으면 **절대 임의로 판단하여 구현하지 않습니다.**
반드시 사용자에게 질문합니다.

예: 아래 사항은 질문 없이 구현 금지

- API endpoint 경로
- request/response JSON 스키마
- pagination 방식
- filter 조건 정의
- 알림 발송 시점(몇 분 전? 정확히 due_at?)
- 반복 알림 여부

---

### 3.3 최소 의존성 원칙

- 표준 라이브러리 우선
- 외부 라이브러리는 필요한 경우에만 추가
- 무분별한 프레임워크 도입 금지

---

## 4. 코드 구조 규칙 (Layered Architecture)

아래 구조를 기본으로 사용합니다.

- `cmd/api/` : main 엔트리포인트
- `internal/config/` : 환경변수 기반 설정 로딩
- `internal/http/` : HTTP 서버, 라우터, 서버 초기화
- `internal/http/handler/` : HTTP 요청/응답 변환 계층
- `internal/middleware/` : auth/logging/recovery
- `internal/service/` : 비즈니스 로직
- `internal/repository/` : DB 접근 계층 (SQL)
- `internal/model/` : 도메인 모델
- `internal/notification/` : 알림 발송/스케줄링 추상화
- `migrations/` : SQL migration
- `infra/` : Terraform 전용 디렉토리

규칙:

- handler는 repository를 직접 호출하면 안 됩니다.
- repository는 비즈니스 로직을 포함하면 안 됩니다.
- service가 handler와 repository 사이에서 비즈니스 규칙을 담당합니다.

---

## 5. API 설계 규칙

### 5.1 API 스타일

- REST JSON API
- base path: `/api/v1`
- 모든 응답은 JSON
- `Content-Type: application/json`

---

### 5.2 에러 응답 형식

모든 에러는 아래 형식을 따릅니다.

```json
{
  "error": {
    "code": "SOME_ERROR_CODE",
    "message": "Human readable message"
  }
}
```

---

### 5.3 상태코드 사용

- 200 OK
- 201 Created
- 204 No Content
- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found
- 409 Conflict
- 500 Internal Server Error

---

## 6. 인증/인가 규칙

### 6.1 Cognito JWT 인증

- Authorization Header 사용
  - `Authorization: Bearer <JWT>`
- API 서버는 JWKS를 통해 JWT를 로컬 검증해야 함
- 요청마다 Cognito API 호출 금지
- 유저 식별자는 Cognito `sub` claim 사용

---

### 6.2 로컬 개발용 인증 우회

local 환경에서는 개발 편의를 위해 auth bypass를 허용할 수 있습니다.

단, 반드시 아래 조건을 만족해야 합니다.

- `AUTH_DEV_MODE=true` 같은 환경변수로 명시적 활성화
- prod/beta/alpha 환경에서는 절대 활성화되면 안 됨

---

## 7. DB 규칙

### 7.1 DB 기본

- PostgreSQL 호환 SQL 사용 (Aurora PostgreSQL)
- ORM auto-migration 금지
- schema 변경은 반드시 migration으로 관리

---

### 7.2 기본 컬럼 규칙

테이블은 기본적으로 다음을 포함합니다.

- id (UUID 권장)
- created_at
- updated_at

soft delete가 필요하면 `deleted_at`을 추가할 수 있으나,
soft delete는 반드시 사용자 승인 후 도입합니다.

---

### 7.3 멀티테넌시(유저 스코프)

Todo는 반드시 Cognito user_id(sub)로 소유권을 구분해야 합니다.

- 모든 조회/수정/삭제는 user_id 조건이 필수

---

## 8. 알림(Notification) 규칙

### 8.1 알림 스케줄링 방식

- due_at 기반 알림은 EventBridge Scheduler를 사용합니다.
- Todo 생성/수정 시 due_at이 존재하면 scheduler entry 생성/업데이트
- Todo 삭제/완료 처리 시 scheduler entry 삭제 또는 disable

---

### 8.2 로컬 환경 알림

local 환경에서는 실제 이메일/푸시 발송을 금지합니다.

- 실제 발송 대신 로그 출력으로 대체

---

## 9. 환경 구성 규칙

### 9.1 local

- DB: local postgres (docker compose)
- API: 로컬 실행(go run) 또는 docker
- Auth: alpha Cognito를 공용으로 사용 가능
- Scheduler/Lambda/SES/SNS: local에서는 실제 호출 금지

---

### 9.2 alpha (dev)

- 최소 비용 구성
- ECS task 최소 1~2
- Aurora serverless 최소 ACU

---

### 9.3 beta (staging)

- prod와 구조는 동일하게 유지
- 스펙은 최소로 설정

---

### 9.4 prod (release)

- ECS 최소 task 2 이상
- autoscaling 활성화
- Aurora backup retention 설정
- deletion protection 활성화
- CloudWatch alarm 필수

---

### 9.5 환경 공유 정책 (매우 중요)

- local은 alpha Cognito를 공유할 수 있음
- local은 alpha DB를 공유하면 안 됨
- beta는 prod 리소스(DB/Auth/Scheduler)를 절대 공유하면 안 됨

---

## 10. 보안 규칙

- 모든 비밀정보는 환경변수로 관리
- 소스코드에 비밀키/토큰/비밀번호 하드코딩 금지
- `.env` 파일 커밋 금지
- Terraform output에 secret 노출 금지

---

## 11. 로컬 개발 요구사항

필수 도구:

- Go 1.23+
- Docker + Docker Compose
- golangci-lint
- migrate (golang-migrate)

레포지토리에 포함되어야 할 것:

- `docker-compose.yml` (postgres)
- `.env.example`
- `Makefile`

---

## 12. CI/CD 규칙 (GitHub Actions)

### 12.1 PR 단계

- `go test ./...`
- `golangci-lint run`
- `terraform fmt -check`
- `terraform validate`
- `terraform plan` (alpha/beta/prod)

---

### 12.2 main merge 단계

- Docker build
- ECR push
- terraform apply (alpha)

---

### 12.3 tag/release 단계

- terraform apply (beta)
- manual approval
- terraform apply (prod)
- DB migration 실행

중요:

- migration은 반드시 CI/CD에서만 실행

---

## 13. Terraform 규칙

### 13.1 Terraform 구조

Terraform은 반드시 아래 구조를 유지합니다.

```
infra/
  modules/
  envs/
    alpha/
    beta/
    prod/
```

---

### 13.2 Terraform state 분리

- alpha/beta/prod는 각각 별도의 tfstate를 사용
- S3 backend + DynamoDB lock을 사용

---

### 13.3 비용 최소화 전략

alpha/beta는 비용 최소화를 위해:

- Aurora serverless min ACU 최소화
- ECS 최소 task 수 유지
- 불필요한 NAT Gateway는 피함 (단, prod는 보안 우선)

---

## 14. 코드 품질 규칙

- 모든 코드는 `gofmt` 적용
- 의미 있는 단위 테스트 작성 (service/repository 중심)
- structured logging 적용

### 14.1 TDD 개발 방식 (everything-claude-code Go 규칙 필수 적용)

신규 기능 개발 또는 기존 기능 수정 시 반드시 **everything-claude-code 플러그인에 내장된 Go 전용 TDD 개발 방식**을 따릅니다.

필수 규칙:

- 구현보다 테스트를 먼저 작성합니다. (Red → Green → Refactor)
- service/repository 계층은 반드시 단위 테스트를 포함해야 합니다.
- handler 계층은 최소한의 통합 테스트 또는 HTTP 테스트를 포함해야 합니다.
- 테스트가 없는 기능 추가/수정은 금지합니다.
- 기능 변경 시 기존 테스트를 먼저 수정하거나 실패하게 만든 뒤 구현을 수정합니다.
- 테스트 실행은 `go test ./...` 기준으로 전체 패키지에서 통과해야 합니다.

Agent는 코드 변경을 제안할 때 반드시 다음 순서를 지켜야 합니다:

1. 테스트 추가/수정
2. 테스트 실패 확인(논리적으로)
3. 구현 코드 변경
4. 리팩토링
5. 전체 테스트 통과

---

## 15. 초기 구현 마일스톤 (반드시 순서대로)

### Milestone 1: 프로젝트 스켈레톤

- go module init
- 기본 디렉토리 구조 생성
- Makefile 추가
- docker-compose.yml (postgres)
- `/health` endpoint 구현

---

### Milestone 2: DB + Migration

- migrations 디렉토리 구성
- 기본 테이블 생성 (users, todos)
- repository 계층 구현

---

### Milestone 3: Todo CRUD API

- todo create/update/delete/list
- status update
- filter 지원

---

### Milestone 4: Auth Middleware

- JWT 검증 (JWKS)
- local dev mode bypass

---

### Milestone 5: Notification Scheduling

- scheduler client interface 정의
- local stub 구현
- AWS 구현은 placeholder로 남김

---

### Milestone 6: CI Pipeline

- GitHub Actions workflow 추가
- lint/test/terraform check

---

## 16. 하지 말아야 할 것들

- frontend 구현 금지
- DynamoDB 도입 금지 (요청 전까지)
- Kubernetes(EKS) 도입 금지
- 자체 인증 시스템 구현 금지
- 과도한 마이크로서비스 분리 금지
- 사용자의 승인 없이 prod 인프라 변경 금지

---

## 17. 큰 변경 전 확인해야 할 질문

아래 사항은 구현 전에 반드시 사용자에게 질문해야 합니다.

- soft delete가 필요한가?
- 반복 task(Recurring)가 필요한가?
- 알림 발송 규칙은 정확히 무엇인가?
- pagination은 offset 기반인가 cursor 기반인가?

---

## 18. 결론

이 프로젝트는 AWS ECS Fargate 기반 배포를 목표로 하지만,
초기 구현은 반드시 local 환경에서 완전히 실행 가능해야 합니다.

먼저 local에서 정상 동작하는 API를 구현하고,
이후 alpha/beta/prod 인프라를 Terraform 기반으로 확장합니다.
