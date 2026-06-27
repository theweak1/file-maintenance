```mermaid
flowchart LR
    subgraph Platform[Platform Layer]
        P1[platform.Current]
        P2[ShowCritical]
        P3[EnsureConfig]
        P4[DefaultConfigDir / DefaultLogDir]
        P5[AvailableBytes]
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
        W2[Unbuffered job input]
        W3[In-memory batch<br/>up to queue-size]
        W4[One backup-space check<br/>per batch]
        W5[Single serialized processor]
        W6[Backup if enabled]
        W7[Delete source after safe backup]
    end

    P1 --> Startup
    Startup --> Config
    Config --> Worker
    P3 --> S5
    P5 --> W4
    P2 --> W6
    W1 --> W2 --> W3 --> W4 --> W5 --> W6 --> W7
```
