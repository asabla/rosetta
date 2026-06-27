permit(principal, action == Rosetta::Action::"CreateSandbox", resource) when { principal has name };
permit(principal, action == Rosetta::Action::"ConnectHost", resource) when { context.host != "" };
permit(principal, action == Rosetta::Action::"UseSecret", resource) when { context.secret != "" && context.long_lived == false };
