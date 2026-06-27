permit (
  principal,
  action == Action::"RunBinary",
  resource
) when {
  principal.approved_binaries.contains(resource.path)
};
