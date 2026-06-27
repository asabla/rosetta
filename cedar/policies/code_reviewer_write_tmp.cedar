permit (
  principal,
  action == Action::"WritePath",
  resource
) when {
  principal.role == "code-reviewer" &&
  principal.assigned_repo == context.repo &&
  resource.repo == principal.assigned_repo &&
  (
    resource.path like "/tmp/*" ||
    resource.path == "/tmp" ||
    resource.path like "/workspace/*/.agent-tmp/*"
  )
};
