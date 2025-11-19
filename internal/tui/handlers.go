package tui

import (
	"fmt"

	"sec-agent/internal/app"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/dbos-inc/dbos-transact-golang/dbos"
)

// listWorkflows returns a command that lists all workflows
func (m App) listWorkflows() tea.Cmd {
	return func() tea.Msg {
		fmt.Println("listing workflows")
		workflows, err := dbos.ListWorkflows(m.dbosCtx)
		if err != nil {
			fmt.Printf("error listing workflows: %v\n", err)
			return errorMsg{err: fmt.Errorf("error listing workflows: %w", err)}
		}

		fmt.Printf("workflows: %v\n", workflows)
		return workflows
	}
}

// startScanWorkflow returns a command that starts the scan workflow
func (m App) startScanWorkflow() tea.Cmd {
	return func() tea.Msg {
		handle, err := dbos.RunWorkflow(m.dbosCtx, app.ScanWorkflow, "")
		if err != nil {
			return scanResultMsg{err: fmt.Errorf("failed to start scan workflow: %w", err)}
		}

		result, err := handle.GetResult()
		if err != nil {
			return scanResultMsg{err: fmt.Errorf("scan workflow failed: %w", err)}
		}

		return scanResultMsg{result: result, err: nil}
	}
}

// Message types
type errorMsg struct {
	err error
}

type scanResultMsg struct {
	result []string
	err    error
}

// getWorkflowSteps returns a command that gets workflow steps
func (m App) getWorkflowSteps() tea.Cmd {
	return func() tea.Msg {
		steps, err := dbos.GetWorkflowSteps(m.dbosCtx, m.selectedWorkflowID)
		if err != nil {
			return errorMsg{err: fmt.Errorf("error getting workflow steps: %w", err)}
		}

		return steps
	}
}

// reportsMsg is a message type for reports list
type reportsMsg struct {
	reports []*app.Report
	err     error
}

// listReportsPendingForApproval returns a command that lists reports pending for approval
func (m App) listReportsPendingForApproval() tea.Cmd {
	return func() tea.Msg {
		reports, err := app.GetReportsPendingForApproval()
		if err != nil {
			return errorMsg{err: fmt.Errorf("error listing reports pending for approval: %w", err)}
		}

		return reportsMsg{reports: reports, err: nil}
	}
}

// startIssueWorkflow returns a command that starts the issue workflow for a report
// The workflow will create the issue and wait for approval
func (m App) startIssueWorkflow(reportID int) tea.Cmd {
	return func() tea.Msg {
		input := app.IssueWorkflowInput{ReportID: reportID}
		handle, err := dbos.RunWorkflow(m.dbosCtx, app.IssueWorkflow, input)
		if err != nil {
			return issueWorkflowStartedMsg{err: fmt.Errorf("failed to start issue workflow: %w", err)}
		}

		// Get workflow ID immediately
		workflowID := handle.GetWorkflowID()

		// Don't wait for result - workflow is waiting for approval
		// Return the workflow ID so we can send approval message later
		return issueWorkflowStartedMsg{workflowID: workflowID, err: nil}
	}
}

// sendIssueApproval sends an approval/rejection message to the workflow
func (m App) sendIssueApproval(workflowID string, approved bool) tea.Cmd {
	return func() tea.Msg {
		status := "rejected"
		if approved {
			status = "approved"
		}

		err := dbos.Send(m.dbosCtx, workflowID, status, "ISSUE_APPROVAL")
		if err != nil {
			return issueResultMsg{err: fmt.Errorf("failed to send approval: %w", err)}
		}

		return issueResultMsg{result: fmt.Sprintf("Issue %s", status), err: nil}
	}
}

// issueWorkflowStartedMsg is sent when the workflow starts
type issueWorkflowStartedMsg struct {
	workflowID string
	err        error
}

// issueApprovalReadyMsg is sent when the issue is ready for approval
type issueApprovalReadyMsg struct {
	issue *app.Issue
	err   error
}

// issueResultMsg is a message type for issue workflow result
type issueResultMsg struct {
	result string
	err    error
}

// listAllIssues returns a command that lists all issues from the database
func (m App) listAllIssues() tea.Cmd {
	return func() tea.Msg {
		issues, err := app.GetAllIssues()
		if err != nil {
			return errorMsg{err: fmt.Errorf("error listing issues: %w", err)}
		}

		return issuesMsg{issues: issues, err: nil}
	}
}

// issuesMsg is a message type for issues list
type issuesMsg struct {
	issues []*app.Issue
	err    error
}

// issueLoadedMsg is a message type for when an issue is loaded by ID
type issueLoadedMsg struct {
	issue *app.Issue
	err   error
}

// loadIssueByID returns a command that loads an issue by its ID
func (m App) loadIssueByID(issueID int) tea.Cmd {
	return func() tea.Msg {
		issue, err := app.GetIssueByID(issueID)
		if err != nil {
			return issueLoadedMsg{err: fmt.Errorf("error loading issue: %w", err)}
		}
		return issueLoadedMsg{issue: issue, err: nil}
	}
}

// renderMarkdown renders markdown content to ANSI-colored text for terminal display
func renderMarkdown(markdown string, width int) (string, error) {
	// Create a renderer with auto-detected style based on terminal
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	// Render the markdown to ANSI-colored text
	out, err := renderer.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return out, nil
}
