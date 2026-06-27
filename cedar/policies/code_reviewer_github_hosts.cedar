permit (
  principal,
  action == Action::"ConnectHost",
  resource in HostGroup::"github"
) when {
  principal.role == "code-reviewer" &&
  principal.assigned_repo == context.repo &&
  principal.approved_task &&
  context.approval_status == "approved"
};
