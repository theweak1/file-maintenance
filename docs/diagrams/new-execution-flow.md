```mermaid
flowchart LR
    subgraph Platform[Platform Layer]
        P1[platform.Current]
        P2[ShowCritical]
        P3[EnsureConfig]
        P4[DefaultConfigDir / DefaultLogDir]
    end

    subgraph Startup[Startup]
        S1[Resolve exe dir]
        S2[Use portable config/log defaults]
        S3[Parse flags]
        S4[Init logger]
        S5[Ensure config]
    end

    subgraph Config[Configuration]
        C1[Read config.ini]
        C2[Parse backup path]
        C3[Parse paths]
        C4[Apply optional settings]
    end

    subgraph Worker[Maintenance Worker]
        W1[Bounded walkers]
        W2[Single file processor]
        W3[Backup if enabled]
        W4[Delete source]
        W5[Cleanup empty dirs]
    end

    P1 --> Startup
    Startup --> Config
    Config --> Worker
    P3 --> S5
    P2 --> W3
```
