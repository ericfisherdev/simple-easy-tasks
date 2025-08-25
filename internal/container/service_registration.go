package container

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/config"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
	"simple-easy-tasks/internal/services"
)

// ServiceNames contains constants for service names used in DI container
const (
	ConfigService                       = "config"
	UserRepositoryService               = "user_repository"
	ProjectRepositoryService            = "project_repository"
	TaskRepositoryService               = "task_repository"
	CommentRepositoryService            = "comment_repository"
	TokenBlacklistRepositoryService     = "token_blacklist_repository"
	PasswordResetTokenRepositoryService = "password_reset_token_repository"
	AuthService                         = "auth_service"
	UserService                         = "user_service"
	ProjectService                      = "project_service"
	TaskService                         = "task_service"
	CommentService                      = "comment_service"
	HealthService                       = "health_service"
)

// RegisterServices registers all application services with the DI container
func RegisterServices(container Container, cfg config.Config, app core.App) error {
	// Register config as singleton
	err := container.RegisterSingleton(ConfigService, func(ctx context.Context, c Container) (interface{}, error) {
		return cfg, nil
	})
	if err != nil {
		return fmt.Errorf("failed to register config service: %w", err)
	}

	// Register repositories
	if err := registerRepositories(container, app); err != nil {
		return fmt.Errorf("failed to register repositories: %w", err)
	}

	// Register services
	if err := registerBusinessServices(container); err != nil {
		return fmt.Errorf("failed to register business services: %w", err)
	}

	return nil
}

// registerRepositories registers all repository implementations
func registerRepositories(container Container, app core.App) error {
	// User Repository
	err := container.RegisterSingleton(UserRepositoryService, func(ctx context.Context, c Container) (interface{}, error) {
		return repository.NewPocketBaseUserRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register user repository: %w", err)
	}

	// Project Repository
	err = container.RegisterSingleton(ProjectRepositoryService, func(ctx context.Context, c Container) (interface{}, error) {
		return repository.NewPocketBaseProjectRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register project repository: %w", err)
	}

	// Task Repository
	err = container.RegisterSingleton(TaskRepositoryService, func(ctx context.Context, c Container) (interface{}, error) {
		return repository.NewPocketBaseTaskRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register task repository: %w", err)
	}

	// Comment Repository
	err = container.RegisterSingleton(CommentRepositoryService, func(ctx context.Context, c Container) (interface{}, error) {
		return repository.NewPocketBaseCommentRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register comment repository: %w", err)
	}

	// Token Blacklist Repository
	err = container.RegisterSingleton(TokenBlacklistRepositoryService, func(ctx context.Context, c Container) (interface{}, error) {
		return repository.NewPocketBaseTokenBlacklistRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register token blacklist repository: %w", err)
	}

	// Password Reset Token Repository
	err = container.RegisterSingleton(PasswordResetTokenRepositoryService, func(ctx context.Context, c Container) (interface{}, error) {
		return repository.NewPocketBasePasswordResetTokenRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register password reset token repository: %w", err)
	}

	return nil
}

// registerBusinessServices registers all business logic services
func registerBusinessServices(container Container) error {
	// Auth Service
	err := container.RegisterSingleton(AuthService, func(ctx context.Context, c Container) (interface{}, error) {
		userRepo, err := c.ResolveWithContext(ctx, UserRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user repository: %w", err)
		}

		blacklistRepo, err := c.ResolveWithContext(ctx, TokenBlacklistRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve token blacklist repository: %w", err)
		}

		resetTokenRepo, err := c.ResolveWithContext(ctx, PasswordResetTokenRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve password reset token repository: %w", err)
		}

		cfg, err := c.ResolveWithContext(ctx, ConfigService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config: %w", err)
		}

		return services.NewAuthService(
			userRepo.(repository.UserRepository),
			blacklistRepo.(domain.TokenBlacklistRepository),
			resetTokenRepo.(domain.PasswordResetTokenRepository),
			cfg.(config.SecurityConfig),
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register auth service: %w", err)
	}

	// User Service
	err = container.RegisterSingleton(UserService, func(ctx context.Context, c Container) (interface{}, error) {
		userRepo, err := c.ResolveWithContext(ctx, UserRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user repository: %w", err)
		}

		authService, err := c.ResolveWithContext(ctx, AuthService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve auth service: %w", err)
		}

		return services.NewUserService(
			userRepo.(repository.UserRepository),
			authService.(services.AuthService),
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register user service: %w", err)
	}

	// Project Service
	err = container.RegisterSingleton(ProjectService, func(ctx context.Context, c Container) (interface{}, error) {
		projectRepo, err := c.ResolveWithContext(ctx, ProjectRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project repository: %w", err)
		}

		userRepo, err := c.ResolveWithContext(ctx, UserRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user repository: %w", err)
		}

		return services.NewProjectService(
			projectRepo.(repository.ProjectRepository),
			userRepo.(repository.UserRepository),
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register project service: %w", err)
	}

	// Task Service
	err = container.RegisterSingleton(TaskService, func(ctx context.Context, c Container) (interface{}, error) {
		taskRepo, err := c.ResolveWithContext(ctx, TaskRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve task repository: %w", err)
		}

		projectRepo, err := c.ResolveWithContext(ctx, ProjectRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project repository: %w", err)
		}

		userRepo, err := c.ResolveWithContext(ctx, UserRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user repository: %w", err)
		}

		return services.NewTaskService(
			taskRepo.(repository.TaskRepository),
			projectRepo.(repository.ProjectRepository),
			userRepo.(repository.UserRepository),
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register task service: %w", err)
	}

	// Comment Service
	err = container.RegisterSingleton(CommentService, func(ctx context.Context, c Container) (interface{}, error) {
		commentRepo, err := c.ResolveWithContext(ctx, CommentRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve comment repository: %w", err)
		}

		taskRepo, err := c.ResolveWithContext(ctx, TaskRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve task repository: %w", err)
		}

		userRepo, err := c.ResolveWithContext(ctx, UserRepositoryService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user repository: %w", err)
		}

		return services.NewCommentService(
			commentRepo.(repository.CommentRepository),
			taskRepo.(repository.TaskRepository),
			userRepo.(repository.UserRepository),
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register comment service: %w", err)
	}

	// Health Service
	err = container.RegisterSingleton(HealthService, func(ctx context.Context, c Container) (interface{}, error) {
		cfg, err := c.ResolveWithContext(ctx, ConfigService)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config: %w", err)
		}

		return services.NewHealthService(cfg.(config.Config)), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register health service: %w", err)
	}

	return nil
}

// ResolveAuthService resolves the auth service from the container
func ResolveAuthService(container Container) (services.AuthService, error) {
	service, err := container.Resolve(AuthService)
	if err != nil {
		return nil, err
	}
	return service.(services.AuthService), nil
}

// ResolveUserService resolves the user service from the container
func ResolveUserService(container Container) (services.UserService, error) {
	service, err := container.Resolve(UserService)
	if err != nil {
		return nil, err
	}
	return service.(services.UserService), nil
}

// ResolveProjectService resolves the project service from the container
func ResolveProjectService(container Container) (services.ProjectService, error) {
	service, err := container.Resolve(ProjectService)
	if err != nil {
		return nil, err
	}
	return service.(services.ProjectService), nil
}

// ResolveTaskService resolves the task service from the container
func ResolveTaskService(container Container) (services.TaskService, error) {
	service, err := container.Resolve(TaskService)
	if err != nil {
		return nil, err
	}
	return service.(services.TaskService), nil
}

// ResolveCommentService resolves the comment service from the container
func ResolveCommentService(container Container) (services.CommentService, error) {
	service, err := container.Resolve(CommentService)
	if err != nil {
		return nil, err
	}
	return service.(services.CommentService), nil
}

// ResolveHealthService resolves the health service from the container
func ResolveHealthService(container Container) (services.HealthServiceInterface, error) {
	service, err := container.Resolve(HealthService)
	if err != nil {
		return nil, err
	}
	return service.(services.HealthServiceInterface), nil
}
