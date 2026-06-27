permit (
  principal,
  action == Action::"UseSecret",
  resource == Secret::"github-readonly-token"
) when {
  principal.role == "code-reviewer" &&
  principal.assigned_repo == context.repo &&
  context.approval_status == "approved"
};
