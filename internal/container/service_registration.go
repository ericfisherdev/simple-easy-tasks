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
	// GitHub repositories
	GitHubIntegrationRepositoryService  = "github_integration_repository"
	GitHubOAuthStateRepositoryService   = "github_oauth_state_repository"
	GitHubIssueMappingRepositoryService = "github_issue_mapping_repository"
	GitHubCommitLinkRepositoryService   = "github_commit_link_repository"
	GitHubPRMappingRepositoryService    = "github_pr_mapping_repository"
	GitHubWebhookEventRepositoryService = "github_webhook_event_repository"
	// Services
	AuthService    = "auth_service"
	UserService    = "user_service"
	ProjectService = "project_service"
	TaskService    = "task_service"
	CommentService = "comment_service"
	HealthService  = "health_service"
	// GitHub services
	GitHubOAuthService   = "github_oauth_service"
	GitHubService        = "github_service"
	GitHubWebhookService = "github_webhook_service"
)

// resolveCommonRepositories resolves commonly used repositories
// resolveAndCast resolves a service and casts it to the expected type
func resolveAndCast[T any](
	ctx context.Context,
	c Container,
	serviceName string,
	errorPrefix string,
) (T, error) {
	var zero T
	service, err := c.ResolveWithContext(ctx, serviceName)
	if err != nil {
		return zero, fmt.Errorf("failed to resolve %s: %w", errorPrefix, err)
	}

	typed, ok := service.(T)
	if !ok {
		return zero, fmt.Errorf("failed to cast %s to correct type", errorPrefix)
	}

	return typed, nil
}

// resolveUserAndAuthServices resolves user repository and auth service dependencies
func resolveUserAndAuthServices(
	ctx context.Context,
	c Container,
) (repository.UserRepository, services.AuthService, error) {
	userRepo, err := resolveAndCast[repository.UserRepository](
		ctx, c, UserRepositoryService, "user repository")
	if err != nil {
		return nil, nil, err
	}

	authService, err := resolveAndCast[services.AuthService](
		ctx, c, AuthService, "auth service")
	if err != nil {
		return nil, nil, err
	}

	return userRepo, authService, nil
}

// resolveProjectAndUserRepos resolves project and user repository dependencies
func resolveProjectAndUserRepos(
	ctx context.Context,
	c Container,
) (repository.ProjectRepository, repository.UserRepository, error) {
	projectRepo, err := resolveAndCast[repository.ProjectRepository](
		ctx, c, ProjectRepositoryService, "project repository")
	if err != nil {
		return nil, nil, err
	}

	userRepo, err := resolveAndCast[repository.UserRepository](
		ctx, c, UserRepositoryService, "user repository")
	if err != nil {
		return nil, nil, err
	}

	return projectRepo, userRepo, nil
}

func resolveCommonRepositories(
	ctx context.Context,
	c Container,
) (repository.TaskRepository, repository.ProjectRepository, repository.UserRepository, error) {
	taskRepo, err := c.ResolveWithContext(ctx, TaskRepositoryService)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve task repository: %w", err)
	}

	projectRepo, err := c.ResolveWithContext(ctx, ProjectRepositoryService)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve project repository: %w", err)
	}

	userRepo, err := c.ResolveWithContext(ctx, UserRepositoryService)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve user repository: %w", err)
	}

	taskRepoTyped, ok := taskRepo.(repository.TaskRepository)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast task repository to correct type")
	}

	projectRepoTyped, ok := projectRepo.(repository.ProjectRepository)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast project repository to correct type")
	}

	userRepoTyped, ok := userRepo.(repository.UserRepository)
	if !ok {
		return nil, nil, nil, fmt.Errorf("failed to cast user repository to correct type")
	}

	return taskRepoTyped, projectRepoTyped, userRepoTyped, nil
}

// RegisterServices registers all application services with the DI container
func RegisterServices(container Container, cfg config.Config, app core.App) error {
	// Register config as singleton
	err := container.RegisterSingleton(ConfigService, func(_ context.Context, _ Container) (interface{}, error) {
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
	err := container.RegisterSingleton(UserRepositoryService, func(_ context.Context, _ Container) (interface{}, error) {
		return repository.NewPocketBaseUserRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register user repository: %w", err)
	}

	// Project Repository
	err = container.RegisterSingleton(ProjectRepositoryService, func(_ context.Context, _ Container) (interface{}, error) {
		return repository.NewPocketBaseProjectRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register project repository: %w", err)
	}

	// Task Repository
	err = container.RegisterSingleton(TaskRepositoryService, func(_ context.Context, _ Container) (interface{}, error) {
		return repository.NewPocketBaseTaskRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register task repository: %w", err)
	}

	// Comment Repository
	err = container.RegisterSingleton(CommentRepositoryService, func(_ context.Context, _ Container) (interface{}, error) {
		return repository.NewPocketBaseCommentRepository(app), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register comment repository: %w", err)
	}

	// Token Blacklist Repository
	err = container.RegisterSingleton(
		TokenBlacklistRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseTokenBlacklistRepository(app), nil
		})
	if err != nil {
		return fmt.Errorf("failed to register token blacklist repository: %w", err)
	}

	// Password Reset Token Repository
	err = container.RegisterSingleton(
		PasswordResetTokenRepositoryService,
		func(_ context.Context, c Container) (interface{}, error) {
			cfgService, cfgErr := resolveAndCast[config.SecurityConfig](
				context.Background(), c, ConfigService, "config service")
			if cfgErr != nil {
				return nil, cfgErr
			}
			return repository.NewPocketBasePasswordResetTokenRepository(app, cfgService.GetPasswordResetSecret()), nil
		})
	if err != nil {
		return fmt.Errorf("failed to register password reset token repository: %w", err)
	}

	// GitHub repositories
	if err := registerGitHubRepositories(container, app); err != nil {
		return fmt.Errorf("failed to register GitHub repositories: %w", err)
	}

	return nil
}

// registerAuthService registers the authentication service
func registerAuthService(container Container) error {
	// Auth Service
	err := container.RegisterSingleton(AuthService, func(ctx context.Context, c Container) (interface{}, error) {
		userRepo, userErr := c.ResolveWithContext(ctx, UserRepositoryService)
		if userErr != nil {
			return nil, fmt.Errorf("failed to resolve user repository: %w", userErr)
		}

		blacklistRepo, blacklistErr := c.ResolveWithContext(ctx, TokenBlacklistRepositoryService)
		if blacklistErr != nil {
			return nil, fmt.Errorf("failed to resolve token blacklist repository: %w", blacklistErr)
		}

		resetTokenRepo, resetErr := c.ResolveWithContext(ctx, PasswordResetTokenRepositoryService)
		if resetErr != nil {
			return nil, fmt.Errorf("failed to resolve password reset token repository: %w", resetErr)
		}

		cfg, cfgErr := c.ResolveWithContext(ctx, ConfigService)
		if cfgErr != nil {
			return nil, fmt.Errorf("failed to resolve config: %w", cfgErr)
		}

		userRepoTyped, ok := userRepo.(repository.UserRepository)
		if !ok {
			return nil, fmt.Errorf("failed to cast user repository to correct type")
		}

		blacklistRepoTyped, ok := blacklistRepo.(domain.TokenBlacklistRepository)
		if !ok {
			return nil, fmt.Errorf("failed to cast blacklist repository to correct type")
		}

		resetTokenRepoTyped, ok := resetTokenRepo.(domain.PasswordResetTokenRepository)
		if !ok {
			return nil, fmt.Errorf("failed to cast reset token repository to correct type")
		}

		cfgTyped, ok := cfg.(config.SecurityConfig)
		if !ok {
			return nil, fmt.Errorf("failed to cast config to security config type")
		}

		return services.NewAuthService(
			userRepoTyped,
			blacklistRepoTyped,
			resetTokenRepoTyped,
			cfgTyped,
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register auth service: %w", err)
	}
	return nil
}

// registerUserService registers the user service
func registerUserService(container Container) error {
	// User Service
	err := container.RegisterSingleton(UserService, func(ctx context.Context, c Container) (interface{}, error) {
		userRepo, authService, err := resolveUserAndAuthServices(ctx, c)
		if err != nil {
			return nil, err
		}

		return services.NewUserService(userRepo, authService), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register user service: %w", err)
	}

	return nil
}

// registerProjectService registers the project service
func registerProjectService(container Container) error {
	// Project Service
	err := container.RegisterSingleton(ProjectService, func(ctx context.Context, c Container) (interface{}, error) {
		projectRepo, userRepo, err := resolveProjectAndUserRepos(ctx, c)
		if err != nil {
			return nil, err
		}

		return services.NewProjectService(projectRepo, userRepo), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register project service: %w", err)
	}

	return nil
}

// registerTaskService registers the task service
func registerTaskService(container Container) error {
	// Task Service
	err := container.RegisterSingleton(TaskService, func(ctx context.Context, c Container) (interface{}, error) {
		taskRepo, projectRepo, userRepo, err := resolveCommonRepositories(ctx, c)
		if err != nil {
			return nil, err
		}

		return services.NewTaskService(taskRepo, projectRepo, userRepo), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register task service: %w", err)
	}

	return nil
}

// registerCommentService registers the comment service
func registerCommentService(container Container) error {
	// Comment Service
	err := container.RegisterSingleton(CommentService, func(ctx context.Context, c Container) (interface{}, error) {
		commentRepo, commentErr := c.ResolveWithContext(ctx, CommentRepositoryService)
		if commentErr != nil {
			return nil, fmt.Errorf("failed to resolve comment repository: %w", commentErr)
		}

		taskRepo, _, userRepo, repoErr := resolveCommonRepositories(ctx, c)
		if repoErr != nil {
			return nil, repoErr
		}

		commentRepoTyped, ok := commentRepo.(repository.CommentRepository)
		if !ok {
			return nil, fmt.Errorf("failed to cast comment repository to correct type")
		}

		return services.NewCommentService(
			commentRepoTyped,
			taskRepo,
			userRepo,
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register comment service: %w", err)
	}

	return nil
}

// registerHealthService registers the health service
func registerHealthService(container Container) error {
	// Health Service
	err := container.RegisterSingleton(HealthService, func(ctx context.Context, c Container) (interface{}, error) {
		cfg, cfgErr := c.ResolveWithContext(ctx, ConfigService)
		if cfgErr != nil {
			return nil, fmt.Errorf("failed to resolve config: %w", cfgErr)
		}

		cfgTyped, ok := cfg.(config.Config)
		if !ok {
			return nil, fmt.Errorf("failed to cast config to correct type")
		}

		return services.NewHealthService(cfgTyped), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register health service: %w", err)
	}

	return nil
}

// registerBusinessServices registers all business logic services
func registerBusinessServices(container Container) error {
	if err := registerAuthService(container); err != nil {
		return err
	}
	if err := registerUserService(container); err != nil {
		return err
	}
	if err := registerProjectService(container); err != nil {
		return err
	}
	if err := registerTaskService(container); err != nil {
		return err
	}
	if err := registerCommentService(container); err != nil {
		return err
	}
	if err := registerHealthService(container); err != nil {
		return err
	}
	if err := registerGitHubServices(container); err != nil {
		return err
	}
	return nil
}

// ResolveAuthService resolves the auth service from the container
func ResolveAuthService(container Container) (services.AuthService, error) {
	service, err := container.Resolve(AuthService)
	if err != nil {
		return nil, err
	}
	serviceTyped, ok := service.(services.AuthService)
	if !ok {
		return nil, fmt.Errorf("failed to cast service to AuthService")
	}
	return serviceTyped, nil
}

// ResolveUserService resolves the user service from the container
func ResolveUserService(container Container) (services.UserService, error) {
	service, err := container.Resolve(UserService)
	if err != nil {
		return nil, err
	}
	serviceTyped, ok := service.(services.UserService)
	if !ok {
		return nil, fmt.Errorf("failed to cast service to UserService")
	}
	return serviceTyped, nil
}

// ResolveProjectService resolves the project service from the container
func ResolveProjectService(container Container) (services.ProjectService, error) {
	service, err := container.Resolve(ProjectService)
	if err != nil {
		return nil, err
	}
	serviceTyped, ok := service.(services.ProjectService)
	if !ok {
		return nil, fmt.Errorf("failed to cast service to ProjectService")
	}
	return serviceTyped, nil
}

// ResolveTaskService resolves the task service from the container
func ResolveTaskService(container Container) (services.TaskService, error) {
	service, err := container.Resolve(TaskService)
	if err != nil {
		return nil, err
	}
	serviceTyped, ok := service.(services.TaskService)
	if !ok {
		return nil, fmt.Errorf("failed to cast service to TaskService")
	}
	return serviceTyped, nil
}

// ResolveCommentService resolves the comment service from the container
func ResolveCommentService(container Container) (services.CommentService, error) {
	service, err := container.Resolve(CommentService)
	if err != nil {
		return nil, err
	}
	serviceTyped, ok := service.(services.CommentService)
	if !ok {
		return nil, fmt.Errorf("failed to cast service to CommentService")
	}
	return serviceTyped, nil
}

// ResolveHealthService resolves the health service from the container
func ResolveHealthService(container Container) (services.HealthServiceInterface, error) {
	service, err := container.Resolve(HealthService)
	if err != nil {
		return nil, err
	}
	serviceTyped, ok := service.(services.HealthServiceInterface)
	if !ok {
		return nil, fmt.Errorf("failed to cast service to HealthServiceInterface")
	}
	return serviceTyped, nil
}

// registerGitHubRepositories registers all GitHub-related repositories
func registerGitHubRepositories(container Container, app core.App) error {
	// GitHub Integration Repository
	err := container.RegisterSingleton(
		GitHubIntegrationRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseGitHubIntegrationRepository(app), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register GitHub integration repository: %w", err)
	}

	// GitHub OAuth State Repository
	err = container.RegisterSingleton(
		GitHubOAuthStateRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseGitHubOAuthStateRepository(app), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register GitHub OAuth state repository: %w", err)
	}

	// GitHub Issue Mapping Repository
	err = container.RegisterSingleton(
		GitHubIssueMappingRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseGitHubIssueMappingRepository(app), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register GitHub issue mapping repository: %w", err)
	}

	// GitHub Commit Link Repository
	err = container.RegisterSingleton(
		GitHubCommitLinkRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseGitHubCommitLinkRepository(app), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register GitHub commit link repository: %w", err)
	}

	// GitHub PR Mapping Repository
	err = container.RegisterSingleton(
		GitHubPRMappingRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseGitHubPRMappingRepository(app), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register GitHub PR mapping repository: %w", err)
	}

	// GitHub Webhook Event Repository
	err = container.RegisterSingleton(
		GitHubWebhookEventRepositoryService,
		func(_ context.Context, _ Container) (interface{}, error) {
			return repository.NewPocketBaseGitHubWebhookEventRepository(app), nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register GitHub webhook event repository: %w", err)
	}

	return nil
}

// registerGitHubServices registers all GitHub-related services
func registerGitHubServices(container Container) error {
	// GitHub OAuth Service
	err := container.RegisterSingleton(GitHubOAuthService, func(ctx context.Context, c Container) (interface{}, error) {
		oauthStateRepo, err := resolveAndCast[services.GitHubOAuthStateRepository](
			ctx, c, GitHubOAuthStateRepositoryService, "GitHub OAuth state repository")
		if err != nil {
			return nil, err
		}

		userService, err := resolveAndCast[services.UserService](
			ctx, c, UserService, "user service")
		if err != nil {
			return nil, err
		}

		cfg, cfgErr := resolveAndCast[config.GitHubConfig](
			ctx, c, ConfigService, "config")
		if cfgErr != nil {
			return nil, cfgErr
		}

		clientID := cfg.GetGitHubClientID()
		clientSecret := cfg.GetGitHubClientSecret()
		redirectURL := cfg.GetGitHubRedirectURL()

		return services.NewGitHubOAuthService(
			clientID,
			clientSecret,
			redirectURL,
			oauthStateRepo,
			userService,
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register GitHub OAuth service: %w", err)
	}

	// GitHub Service
	err = container.RegisterSingleton(GitHubService, func(ctx context.Context, c Container) (interface{}, error) {
		integrationRepo, resolveErr := resolveAndCast[services.GitHubIntegrationRepository](
			ctx, c, GitHubIntegrationRepositoryService, "GitHub integration repository")
		if resolveErr != nil {
			return nil, resolveErr
		}

		issueMappingRepo, mappingErr := resolveAndCast[services.GitHubIssueMappingRepository](
			ctx, c, GitHubIssueMappingRepositoryService, "GitHub issue mapping repository")
		if mappingErr != nil {
			return nil, mappingErr
		}

		commitLinkRepo, commitErr := resolveAndCast[services.GitHubCommitLinkRepository](
			ctx, c, GitHubCommitLinkRepositoryService, "GitHub commit link repository")
		if commitErr != nil {
			return nil, commitErr
		}

		prMappingRepo, prErr := resolveAndCast[services.GitHubPRMappingRepository](
			ctx, c, GitHubPRMappingRepositoryService, "GitHub PR mapping repository")
		if prErr != nil {
			return nil, prErr
		}

		cfg, cfgErr := resolveAndCast[config.GitHubConfig](
			ctx, c, ConfigService, "config")
		if cfgErr != nil {
			return nil, cfgErr
		}

		webhookSecret := cfg.GetGitHubWebhookSecret()

		return services.NewGitHubService(
			integrationRepo,
			issueMappingRepo,
			commitLinkRepo,
			prMappingRepo,
			webhookSecret,
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register GitHub service: %w", err)
	}

	// GitHub Webhook Service
	err = container.RegisterSingleton(GitHubWebhookService, func(ctx context.Context, c Container) (interface{}, error) {
		integrationRepo, resolveErr := resolveAndCast[services.GitHubIntegrationRepository](
			ctx, c, GitHubIntegrationRepositoryService, "GitHub integration repository")
		if resolveErr != nil {
			return nil, resolveErr
		}

		webhookEventRepo, webhookErr := resolveAndCast[services.GitHubWebhookEventRepository](
			ctx, c, GitHubWebhookEventRepositoryService, "GitHub webhook event repository")
		if webhookErr != nil {
			return nil, webhookErr
		}

		githubService, serviceErr := resolveAndCast[*services.GitHubService](
			ctx, c, GitHubService, "GitHub service")
		if serviceErr != nil {
			return nil, serviceErr
		}

		taskService, taskErr := resolveAndCast[services.TaskService](
			ctx, c, TaskService, "task service")
		if taskErr != nil {
			return nil, taskErr
		}

		cfg, cfgErr := resolveAndCast[config.GitHubConfig](
			ctx, c, ConfigService, "config")
		if cfgErr != nil {
			return nil, cfgErr
		}

		webhookSecret := cfg.GetGitHubWebhookSecret()

		return services.NewGitHubWebhookService(
			webhookSecret,
			integrationRepo,
			webhookEventRepo,
			githubService,
			taskService,
		), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register GitHub webhook service: %w", err)
	}

	return nil
}
