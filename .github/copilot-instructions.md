# Navidrome Code Guidelines

This is a music streaming server written in Go with a React frontend. The application manages music libraries, provides streaming capabilities, and offers various features like artist information, artwork handling, and external service integrations.

## Code Standards

### Backend (Go)
- Follow standard Go conventions and idioms
- Use context propagation for cancellation signals
- Write unit tests for new functionality using Ginkgo/Gomega
- Use mutex appropriately for concurrent operations
- Implement interfaces for dependencies to facilitate testing

### Frontend (React)
- Use functional components with hooks
- Follow React best practices for state management
- Implement PropTypes for component properties
- Prefer using React-Admin and Material-UI components
- Icons should be imported from `react-icons` only
- Follow existing patterns for API interaction

## Repository Structure
- `core/`: Server-side business logic (artwork handling, playback, etc.)
- `ui/`: React frontend components
- `model/`: Data models and repository interfaces
- `server/`: API endpoints and server implementation
- `utils/`: Shared utility functions
- `persistence/`: Database access layer
- `scanner/`: Music library scanning functionality

## Key Guidelines
1. Maintain cache management patterns for performance
2. Follow the existing concurrency patterns (mutex, atomic)
3. Use the testing framework appropriately (Ginkgo/Gomega for Go)
4. Keep UI components focused and reusable
5. Document configuration options in code
6. Consider performance implications when working with music libraries
7. Follow existing error handling patterns
8. Ensure compatibility with external services (LastFM, Spotify, Deezer)

## Development Workflow
- Test changes thoroughly, especially around concurrent operations
- Validate both backend and frontend interactions
- Consider how changes will affect user experience and performance
- Test with different music library sizes and configurations
- Before committing, ALWAYS run `make format lint test`, and make sure there are no issues

## Important commands
- `make build`: Build the application
- `make test`: Run Go tests
- To run tests for a specific package, use `make test PKG=./pkgname/...`
- `make lintall`: Run linters
- `make format`: Format code
