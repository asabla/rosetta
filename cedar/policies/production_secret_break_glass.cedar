permit (
  principal,
  action == Action::"UseSecret",
  resource
) when {
  resource.production &&
  context.risk == "approved-high" &&
  context.human_approval_id != ""
};
