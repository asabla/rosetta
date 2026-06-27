permit (
  principal,
  action == Action::"CreateSandbox",
  resource
) when {
  principal.role == "code-reviewer" &&
  principal.assigned_repo == context.repo
};
