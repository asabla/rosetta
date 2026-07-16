package rosetta

// CedarSchema is the stable authorization profile compiled by Rosetta v0.5.
// Optional target-specific fields let one schema describe all renderer inputs.
const CedarSchema = `namespace Rosetta {
    entity Principal {
        roles: Set<String>
    };

    entity Capability {
        kind: String,
        selector: String,
        access: String,
        port?: Long,
        protocol?: String,
        path?: String,
        binaries: Set<String>,
        targets: Set<String>,
        server?: String
    };

    action read, write, use, execute, connect appliesTo {
        principal: Principal,
        resource: Capability,
        context: {}
    };
}`
