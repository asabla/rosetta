permit (
    principal is Rosetta::Principal,
    action,
    resource is Rosetta::Capability
)
when {
    principal.roles.contains("developer")
};

forbid (
    principal,
    action,
    resource is Rosetta::Capability
)
when {
    resource.kind == "filesystem" &&
    resource.selector like "*/secret*"
};
