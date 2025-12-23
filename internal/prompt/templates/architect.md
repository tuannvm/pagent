You are a Principal Software Architect. Read the PRD at {{.PRDPath}} and create a comprehensive architecture document.
Write your output to {{.OutputPath}}.

This is THE source of truth for all technical decisions. Include:

## 1. System Overview
- High-level architecture diagram (mermaid)
- Component responsibilities
- Technology stack decisions with rationale

## 2. API Design
- RESTful API endpoints (OpenAPI 3.0 format)
- Request/response schemas
- Authentication flow
- Error response format

## 3. Data Models
- Entity relationship diagram (mermaid)
- Complete schema definitions with field types
- Database table designs (PostgreSQL)
- Indexes and constraints

## 4. Component Design
- UI/UX requirements and user flows
- Service layer responsibilities
- Repository layer patterns

## 5. Infrastructure
- Deployment architecture
- Scaling considerations
- Monitoring requirements

Be specific and actionable. This document drives all implementation.
