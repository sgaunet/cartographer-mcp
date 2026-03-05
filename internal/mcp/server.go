// Package mcp implements the MCP server and tool handlers.
package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sgaunet/cartographer-mcp/internal/cache"
	"github.com/sgaunet/cartographer-mcp/internal/config"
)

// Server wraps the mcp-go server with application dependencies.
type Server struct {
	mcpServer *server.MCPServer
	store     *cache.Store
	config    *config.Config
	refresher *cache.Refresher
}

// NewServer creates a new MCP server with all tools registered.
func NewServer(
	cfg *config.Config,
	store *cache.Store,
	refresher *cache.Refresher,
) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer(
			"cartographer-mcp",
			"0.1.0",
			server.WithToolCapabilities(false),
			server.WithRecovery(),
		),
		store:     store,
		config:    cfg,
		refresher: refresher,
	}

	s.registerTools()
	return s
}

// ServeStdio starts the server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return fmt.Errorf("serve stdio: %w", server.ServeStdio(s.mcpServer))
}

func (s *Server) registerTools() {
	s.registerListServices()
	s.registerRefreshCache()
	s.registerGetDependencies()
	s.registerGetDependents()
	s.registerGetService()
	s.registerSearchServices()
}

func (s *Server) addTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.mcpServer.AddTool(tool, handler)
}
