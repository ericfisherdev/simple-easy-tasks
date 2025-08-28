# Simple Easy Tasks API Documentation

## Overview

This document provides comprehensive documentation for all API endpoints in the Simple Easy Tasks application. The API follows REST principles and returns JSON responses with consistent error handling.

**Base URL**: `http://localhost:8090/api`
**API Version**: 1.0.0
**Authentication**: JWT Bearer tokens

## Table of Contents

1. [Authentication](#authentication)
2. [Error Handling](#error-handling)
3. [Rate Limiting](#rate-limiting)
4. [User Management](#user-management)  
5. [Project Management](#project-management)
6. [Task Management](#task-management)
7. [Real-time Features](#real-time-features)
8. [Health & Monitoring](#health--monitoring)
9. [Response Schemas](#response-schemas)
10. [Security Considerations](#security-considerations)

---

## Authentication

All authentication endpoints are accessible without authentication except for logout and profile endpoints.

### POST /api/auth/login
Authenticate user and receive JWT tokens.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2025-01-15T10:30:00Z"
  }
}
```

**Cookies Set:**
- `access_token` (HttpOnly, Secure)
- `refresh_token` (HttpOnly, Secure, 7 days)

### POST /api/auth/register
Register a new user account.

**Request Body:**
```json
{
  "email": "newuser@example.com",
  "password": "securepassword123",
  "name": "John Doe"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user123",
      "email": "newuser@example.com",
      "name": "John Doe",
      "role": "user",
      "created_at": "2025-01-15T10:30:00Z"
    }
  }
}
```

### POST /api/auth/refresh
Refresh access token using refresh token.

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2025-01-15T11:30:00Z"
  }
}
```

### POST /api/auth/logout
**Authorization Required**

Logout user and invalidate tokens.

**Response (200):**
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

### GET /api/auth/me
**Authorization Required**

Get current user profile.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user123",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "user",
      "avatar": "https://...",
      "preferences": {...}
    }
  }
}
```

### POST /api/auth/forgot-password
Initiate password reset process.

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "If the email exists, a password reset link has been sent"
}
```

### POST /api/auth/reset-password
Reset password using token.

**Request Body:**
```json
{
  "token": "reset-token-here",
  "new_password": "newsecurepassword"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Password has been successfully reset"
}
```

---

## User Management

All user endpoints require authentication.

### GET /api/users/profile
**Authorization Required**

Get current user's profile.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user123",
      "email": "user@example.com",
      "name": "John Doe",
      "avatar": "https://...",
      "preferences": {
        "theme": "dark",
        "notifications": true
      }
    }
  }
}
```

### PUT /api/users/profile
**Authorization Required**

Update user profile.

**Request Body:**
```json
{
  "name": "Updated Name",
  "avatar": "https://newavatar.com/image.jpg",
  "preferences": {
    "theme": "light",
    "notifications": false
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user123",
      "email": "user@example.com",
      "name": "Updated Name",
      "avatar": "https://newavatar.com/image.jpg",
      "preferences": {
        "theme": "light",
        "notifications": false
      }
    }
  }
}
```

### POST /api/users/avatar
**Authorization Required**

Update user avatar.

**Request Body:**
```json
{
  "avatar": "https://example.com/avatar.jpg"
}
```

### DELETE /api/users/avatar
**Authorization Required**

Remove user avatar.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {...}
  }
}
```

### PUT /api/users/preferences
**Authorization Required**

Update user preferences.

**Request Body:**
```json
{
  "theme": "dark",
  "notifications": true,
  "language": "en"
}
```

### GET /api/users
**Admin Authorization Required**

List all users (admin only).

**Query Parameters:**
- `search` - Search by email
- `limit` - Max results (default: 50)
- `offset` - Pagination offset (default: 0)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "users": [...],
    "meta": {
      "total": 100,
      "limit": 50,
      "offset": 0
    }
  }
}
```

### GET /api/users/:id
**Admin Authorization Required**

Get user by ID (admin only).

### PUT /api/users/:id/role
**Admin Authorization Required**

Update user role (admin only).

**Request Body:**
```json
{
  "role": "admin"
}
```

### DELETE /api/users/:id
**Admin Authorization Required**

Delete user (admin only, cannot delete self).

---

## Project Management

All project endpoints require authentication.

### GET /api/projects
**Authorization Required**

List user's projects (owned and member).

**Query Parameters:**
- `limit` - Max results (1-100, default: 50)
- `offset` - Pagination offset (default: 0)
- `status` - Filter by status (active, archived)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "projects": [
      {
        "id": "proj123",
        "title": "My Project",
        "description": "Project description",
        "slug": "my-project",
        "owner_id": "user123",
        "status": "active",
        "color": "#3b82f6",
        "icon": "📊",
        "member_ids": ["user456", "user789"],
        "settings": {
          "is_private": false,
          "allow_guest_view": true,
          "enable_comments": true
        },
        "created_at": "2025-01-15T10:30:00Z",
        "updated_at": "2025-01-15T10:30:00Z"
      }
    ],
    "meta": {
      "total": 5,
      "limit": 50,
      "offset": 0
    }
  }
}
```

### POST /api/projects
**Authorization Required**

Create new project.

**Request Body:**
```json
{
  "title": "New Project",
  "description": "Project description",
  "slug": "new-project",
  "color": "#3b82f6",
  "icon": "📊",
  "settings": {
    "is_private": false,
    "allow_guest_view": true,
    "enable_comments": true
  }
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "project": {
      "id": "proj123",
      "title": "New Project",
      "owner_id": "user123",
      "status": "active",
      "member_ids": [],
      "created_at": "2025-01-15T10:30:00Z"
    }
  }
}
```

### GET /api/projects/:id
**Authorization Required**

Get project by ID (requires access).

### PUT /api/projects/:id
**Owner Authorization Required**

Update project (owner only).

**Request Body:**
```json
{
  "title": "Updated Title",
  "description": "Updated description",
  "color": "#ef4444",
  "settings": {
    "is_private": true
  }
}
```

### DELETE /api/projects/:id
**Owner Authorization Required**

Delete project (owner only).

### POST /api/projects/:id/members
**Owner Authorization Required**

Add member to project (owner only).

**Request Body:**
```json
{
  "user_id": "user456"
}
```

### DELETE /api/projects/:id/members/:memberID
**Owner Authorization Required**

Remove member from project (owner only).

---

## Task Management

All task endpoints require authentication and are nested under projects.

### GET /api/projects/:projectId/tasks
**Authorization Required**

List tasks in project with advanced filtering.

**Query Parameters:**
- `limit` - Max results (1-100, default: 20)
- `offset` - Pagination offset (default: 0)
- `status` - Filter by status (backlog, todo, developing, review, complete)
- `priority` - Filter by priority (low, medium, high, critical)
- `assignee` - Filter by assignee ID
- `reporter` - Filter by reporter ID
- `search` - Full-text search in title/description
- `archived` - Include/exclude archived (true/false)
- `due_before` - Filter tasks due before date (ISO 8601)
- `due_after` - Filter tasks due after date (ISO 8601)
- `sort_by` - Sort field (created, updated, due_date, priority, title)
- `sort_order` - Sort order (asc, desc, default: desc)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "id": "task123",
        "title": "Implement authentication",
        "description": "Add JWT authentication to API",
        "status": "developing",
        "priority": "high",
        "project_id": "proj123",
        "assignee_id": "user456",
        "reporter_id": "user123",
        "due_date": "2025-01-20T00:00:00Z",
        "position": 1,
        "tags": ["backend", "security"],
        "time_spent": 7.5,
        "time_estimated": 12.0,
        "archived": false,
        "parent_task_id": null,
        "subtask_count": 2,
        "comment_count": 3,
        "created_at": "2025-01-15T10:30:00Z",
        "updated_at": "2025-01-15T11:30:00Z"
      }
    ],
    "meta": {
      "total": 25,
      "limit": 20,
      "offset": 0,
      "count": 20
    }
  }
}
```

### POST /api/projects/:projectId/tasks
**Authorization Required**

Create new task in project.

**Request Body:**
```json
{
  "title": "New Task",
  "description": "Task description",
  "priority": "medium",
  "status": "backlog",
  "assignee_id": "user456",
  "due_date": "2025-01-25T00:00:00Z",
  "tags": ["frontend"],
  "time_estimated": 8.0
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "task": {
      "id": "task456",
      "title": "New Task",
      "project_id": "proj123",
      "reporter_id": "user123",
      "status": "backlog",
      "position": 1,
      "created_at": "2025-01-15T12:00:00Z"
    }
  }
}
```

### GET /api/projects/:projectId/tasks/:id
**Authorization Required**

Get task by ID.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "task": {
      "id": "task123",
      "title": "Implement authentication",
      "description": "Add JWT authentication to API",
      "status": "developing",
      "priority": "high",
      "project_id": "proj123",
      "assignee_id": "user456",
      "reporter_id": "user123",
      "due_date": "2025-01-20T00:00:00Z",
      "position": 1,
      "tags": ["backend", "security"],
      "time_spent": 7.5,
      "time_estimated": 12.0,
      "archived": false,
      "parent_task_id": null,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T11:30:00Z"
    }
  }
}
```

### PUT /api/projects/:projectId/tasks/:id
**Authorization Required**

Update task.

**Request Body:**
```json
{
  "title": "Updated title",
  "description": "Updated description",
  "priority": "critical",
  "status": "review",
  "assignee_id": "user789",
  "due_date": "2025-01-22T00:00:00Z",
  "tags": ["backend", "security", "urgent"]
}
```

### DELETE /api/projects/:projectId/tasks/:id
**Authorization Required**

Delete task.

**Response (200):**
```json
{
  "success": true,
  "message": "Task deleted successfully"
}
```

### POST /api/projects/:projectId/tasks/:id/move
**Authorization Required**

Move task to different status/position.

**Request Body:**
```json
{
  "new_status": "review",
  "new_position": 2
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "task": {...}
  },
  "message": "Task moved successfully"
}
```

### PUT /api/projects/:projectId/tasks/:id/status
**Authorization Required**

Update task status only.

**Request Body:**
```json
{
  "status": "complete"
}
```

### PUT /api/projects/:projectId/tasks/:id/position
**Authorization Required**

Update task position within current column.

**Request Body:**
```json
{
  "position": 3
}
```

### POST /api/projects/:projectId/tasks/:id/assign
**Authorization Required**

Assign task to user.

**Request Body:**
```json
{
  "assignee_id": "user456"
}
```

### DELETE /api/projects/:projectId/tasks/:id/assign
**Authorization Required**

Unassign task.

### POST /api/projects/:projectId/tasks/:id/duplicate
**Authorization Required**

Duplicate task with options.

**Request Body:**
```json
{
  "include_subtasks": false,
  "include_comments": false,
  "include_attachments": false,
  "reset_progress": true,
  "reset_time_spent": true
}
```

### GET /api/projects/:projectId/tasks/:id/history
**Authorization Required**

Get task history/changelog.

**Query Parameters:**
- `limit` - Max results (1-100, default: 50)
- `offset` - Pagination offset (default: 0)

**Response (200):**
```json
{
  "success": true,
  "data": {
    "history": [
      {
        "id": "history-1",
        "task_id": "task123",
        "user_id": "user123",
        "action": "created",
        "changes": {},
        "created_at": "2025-01-15T10:30:00Z"
      }
    ],
    "meta": {
      "total": 1,
      "limit": 50,
      "offset": 0
    }
  }
}
```

### POST /api/projects/:projectId/tasks/:id/time-log
**Authorization Required**

Log time spent on task.

**Request Body:**
```json
{
  "hours": 2.5,
  "description": "Implemented authentication logic",
  "logged_at": "2025-01-15T14:00:00Z"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "time_log": {
      "id": "log-123",
      "task_id": "task123",
      "user_id": "user123",
      "hours": 2.5,
      "description": "Implemented authentication logic",
      "logged_at": "2025-01-15T14:00:00Z",
      "created_at": "2025-01-15T14:00:00Z"
    },
    "task": {
      "time_spent": 10.0
    }
  },
  "message": "Time logged successfully"
}
```

### POST /api/projects/:projectId/tasks/:id/subtasks
**Authorization Required**

Create subtask.

**Request Body:**
```json
{
  "title": "Subtask title",
  "description": "Subtask description",
  "priority": "medium"
}
```

### GET /api/projects/:projectId/tasks/:id/subtasks
**Authorization Required**

List subtasks.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "subtasks": [...],
    "meta": {
      "count": 2,
      "parent_task": "task123",
      "parent_title": "Implement authentication"
    }
  }
}
```

### POST /api/projects/:projectId/tasks/:id/dependencies
**Authorization Required**

Add task dependency.

**Request Body:**
```json
{
  "dependency_id": "task456"
}
```

### DELETE /api/projects/:projectId/tasks/:id/dependencies/:depId
**Authorization Required**

Remove task dependency.

---

## Real-time Features

Real-time endpoints for live updates and subscriptions.

### POST /api/realtime/subscriptions
**Authorization Required**

Create event subscription for real-time updates.

**Request Body:**
```json
{
  "project_id": "proj123",
  "event_types": ["task_created", "task_updated", "task_moved"],
  "filters": {
    "assignee_id": "user123"
  }
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "sub123",
    "user_id": "user123",
    "project_id": "proj123",
    "event_types": ["task_created", "task_updated", "task_moved"],
    "filters": {
      "assignee_id": "user123"
    },
    "created_at": "2025-01-15T12:00:00Z"
  },
  "message": "Subscription created successfully"
}
```

### GET /api/realtime/subscriptions
**Authorization Required**

List user's subscriptions.

### GET /api/realtime/subscriptions/:id
**Authorization Required**

Get specific subscription.

### PUT /api/realtime/subscriptions/:id
**Authorization Required**

Update subscription.

### DELETE /api/realtime/subscriptions/:id
**Authorization Required**

Delete subscription.

### GET /api/realtime/events
**Authorization Required**

Server-Sent Events stream for real-time updates.

**Query Parameters:**
- `project_id` - Filter events by project
- `event_types` - Comma-separated event types

**Headers:**
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

**Event Format:**
```
data: {"type":"task_updated","task_id":"task123","project_id":"proj123","timestamp":"2025-01-15T12:00:00Z","data":{...}}

data: {"type":"connected","subscription_id":"sub123"}

: keepalive
```

### GET /api/realtime/connections
**Authorization Required**

Get active connection information.

### GET /api/realtime/health
**No Authorization Required**

Health check for real-time services.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "active_subscriptions": 25,
    "timestamp": "2025-01-15T12:00:00Z"
  },
  "message": "Real-time service is healthy"
}
```

---

## Health & Monitoring

System health and monitoring endpoints.

### GET /health
**No Authorization Required**

Comprehensive health check.

**Response (200):**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T12:00:00Z",
  "version": "1.0.0",
  "uptime": "24h30m15s",
  "environment": "development",
  "checks": {
    "database": "healthy",
    "redis": "healthy", 
    "external_apis": "healthy"
  }
}
```

### GET /health/live
**No Authorization Required**

Liveness probe - basic application health.

### GET /health/ready
**No Authorization Required**

Readiness probe - ready to serve traffic.

### GET /health/detailed
**No Authorization Required**

Detailed health with system metrics.

### GET /ping
**No Authorization Required**

Simple ping endpoint.

**Response (200):**
```json
{
  "message": "pong",
  "time": 1642248000
}
```

### GET /metrics
**No Authorization Required**

Application metrics for monitoring.

**Response (200):**
```json
{
  "timestamp": 1642248000,
  "system": {
    "environment": "development",
    "version": "1.0.0"
  },
  "rate_limiting": {
    "enabled": true,
    "redis_enabled": true,
    "cache_stats": {
      "hits": 1250,
      "misses": 45,
      "total_requests": 1295
    }
  }
}
```

---

## Error Handling

All API endpoints follow consistent error response format.

### Error Response Structure

```json
{
  "success": false,
  "error": {
    "type": "ERROR_TYPE",
    "code": "SPECIFIC_CODE",
    "message": "Human readable message",
    "details": "Additional details (optional)"
  }
}
```

### Error Types

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid request data |
| `AUTHENTICATION_ERROR` | 401 | Invalid or missing authentication |
| `AUTHORIZATION_ERROR` | 403 | Insufficient permissions |
| `NOT_FOUND_ERROR` | 404 | Resource not found |
| `CONFLICT_ERROR` | 409 | Resource conflict |
| `RATE_LIMIT_ERROR` | 429 | Rate limit exceeded |
| `INTERNAL_SERVER_ERROR` | 500 | Server error |

### Common Error Codes

- `INVALID_REQUEST` - Malformed request body
- `USER_NOT_FOUND` - User not found in context
- `MISSING_PROJECT_ID` - Required project ID missing
- `ACCESS_DENIED` - Insufficient permissions
- `TASK_NOT_FOUND` - Task not found
- `INVALID_STATUS` - Invalid task status
- `RATE_LIMIT_EXCEEDED` - Too many requests

---

## Rate Limiting

Rate limiting is applied globally and can be configured per environment.

**Default Limits:**
- 100 requests per minute per IP
- Sliding window implementation
- Redis-backed for distributed environments

**Headers:**
- `X-RateLimit-Limit` - Request limit
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Reset timestamp

**Rate Limit Exceeded Response (429):**
```json
{
  "success": false,
  "error": {
    "type": "RATE_LIMIT_ERROR",
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again later.",
    "details": "Limit: 100 requests per minute"
  }
}
```

---

## Security Considerations

### Authentication
- JWT tokens with configurable expiration
- Refresh token rotation
- HttpOnly, Secure cookie options
- Token blacklisting on logout

### Authorization
- Role-based access control (admin, user)
- Resource-level permissions (project ownership)
- Request context validation

### Input Validation
- Request schema validation
- SQL injection prevention
- XSS protection
- CSRF tokens for state-changing operations

### Security Headers
- `X-Frame-Options: SAMEORIGIN`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Content-Security-Policy`
- CORS configuration

### Data Protection
- Password hashing (bcrypt)
- Sensitive data sanitization in responses
- Secure password reset flows
- Email enumeration protection

---

## Response Schemas

### User Object
```json
{
  "id": "string",
  "email": "string",
  "name": "string",
  "role": "admin|user",
  "avatar": "string|null",
  "preferences": {
    "theme": "light|dark",
    "notifications": "boolean",
    "language": "string"
  },
  "created_at": "ISO 8601 datetime",
  "updated_at": "ISO 8601 datetime"
}
```

### Project Object
```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "slug": "string",
  "owner_id": "string",
  "status": "active|archived",
  "color": "string (hex)",
  "icon": "string (emoji)",
  "member_ids": ["string"],
  "settings": {
    "is_private": "boolean",
    "allow_guest_view": "boolean",
    "enable_comments": "boolean",
    "custom_fields": "object",
    "notifications": "object"
  },
  "created_at": "ISO 8601 datetime",
  "updated_at": "ISO 8601 datetime"
}
```

### Task Object
```json
{
  "id": "string",
  "title": "string",
  "description": "string",
  "status": "backlog|todo|developing|review|complete",
  "priority": "low|medium|high|critical",
  "project_id": "string",
  "assignee_id": "string|null",
  "reporter_id": "string",
  "due_date": "ISO 8601 datetime|null",
  "position": "integer",
  "tags": ["string"],
  "time_spent": "float (hours)",
  "time_estimated": "float (hours)",
  "archived": "boolean",
  "parent_task_id": "string|null",
  "subtask_count": "integer",
  "comment_count": "integer",
  "created_at": "ISO 8601 datetime",
  "updated_at": "ISO 8601 datetime"
}
```

### Event Subscription Object
```json
{
  "id": "string",
  "user_id": "string", 
  "project_id": "string|null",
  "event_types": ["string"],
  "filters": "object",
  "active": "boolean",
  "created_at": "ISO 8601 datetime",
  "updated_at": "ISO 8601 datetime"
}
```

---

## API Versioning

The API uses URL path versioning:
- Current version: `/api/v1/` (implied as default `/api/`)
- Future versions: `/api/v2/`, `/api/v3/`, etc.

Backward compatibility is maintained for at least one major version.

---

This documentation covers all available endpoints as of API version 1.0.0. For the most up-to-date information, refer to the OpenAPI/Swagger specification at `/api/docs` (when available).