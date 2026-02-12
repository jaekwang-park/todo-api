# todo-api

## 프로젝트 소개
할 일(todo)관리 앱의 백엔드 프로젝트.

## 주요 기능들
- 유저 가입 (SNS연동 혹은 이메일로 등록)
- 유저 정보 편집 (닉네임, 프로필)
- 로그인/로그아웃
- todo 관리 (등록, 변경, 삭제)
- todo 목록 보기 (필터링 포함)
- 상태 변경 (완료처리)
- 알림 (메일, 푸시)

## 기술 스택
- 프로그래밍 언어: Go 1.23+
- DB: Aurora PostgreSQL Serverless v2
- 인증: Cognito User Pool
- 스케줄링: EventBridge Scheduler → Lambda(알림 워커)
- 이메일 발송: Amazon SES
- 푸시 발송: Amazon SNS
- 스토리지: S3
- Scale-out: ECS on Fargate (컨테이너 실행) + ALB(HTTP 라우팅)
- CDN: CloudFront
- CI/CD: GitHub Actions
- 관측/로깅: CloudWatch Logs/Metrics/Alarms
- IaC: Terraform, AWS Provider

## 환경 구성
- local
  - API 서버: 로컬 Go 실행 또는 Docker 실행
  - DB: 로컬 Postgres(Docker)
  - Auth: alpha Cognito 공용
  - 알림: 로컬에서는 발송 금지(mock)
- alpha(dev)
  - ECS Fargate: 최소 task 1~2
  - ALB: 1개
  - Aurora Serverless v2: 최소 ACU 설정
  - Cognito User Pool: 1개
  - Scheduler + Lambda: 구성하되 실제 알림은 sandbox 모드 가능
  - CloudWatch Logs: 기본 활성화
- beta(staging)
  - prod와 거의 동일한 구조 유지
  - 단, 스펙은 낮게(최소 task 수, DB 최소 용량)
- prod(release)
  - ECS Fargate: 최소 task 2 이상 + autoscaling
  - ALB: 1개
  - Aurora Serverless v2: 백업/삭제보호 설정
  - Cognito User Pool: prod 전용
  - Scheduler + Lambda: prod 전용
  - CloudWatch Alarm: CPU/Memory/5xx/DB connections 등 필수

## 로컬 실행
- docker compose로 postgres 실행
- go run으로 API 실행
- 환경변수로 Cognito 설정만 alpha 값 사용

## Terraform 적용 방식 (환경별 분리 강제)
### Terraform 구조
```
infra/
  modules/
    vpc/
    ecs_service/
    aurora/
    cognito/
    lambda_notify/
    scheduler/
  envs/
    alpha/
    beta/
    prod/
```

### state는 반드시 분리
- alpha/beta/prod 각각 다른 tfstate 사용

## CI/CD 파이프라인 (GitHub Actions 기준)
### PR 단계
- go test
- golangci-lint
- terraform fmt / validate
- terraform plan (alpha/beta/prod)

### merge(main) 단계
- Docker build
- ECR push
- terraform apply (alpha)

### tag/release 단계
- terraform apply (beta)
- 승인(Manual Approval)
- terraform apply (prod)
- DB migration 실행

운영 팁: migration은 반드시 CI/CD에서만 실행하도록 통제

자세한 개발 규칙은 CLAUDE.md 참고
