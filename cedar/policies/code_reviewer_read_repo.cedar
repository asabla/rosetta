permit (
  principal,
  action == Action::"ReadPath",
  resource
) when {
  principal.role == "code-reviewer" &&
  principal.assigned_repo == context.repo &&
  resource.repo == principal.assigned_repo &&
  resource.path like "/workspace/*"
};
