```mermaid
flowchart TD
    subgraph Startup[Phase 1: Application Startup]
        A[Parse CLI Arguments]
        B[Resolve Default Paths]
        C[Build AppConfig]
        D[Initialize Logger]
    end

    subgraph SetupCheck[Phase 2: Setup Check]
        E{Config Exists?}
        F[Launch Setup Wizard]
        G[Exit Cancelled]
    end

    subgraph Configuration[Phase 3: Configuration Loading]
        H[Read config.ini<br/>Parse backup & paths]
        I{Any Path Has<br/>Backup Enabled?}
        J[Validate Backup Path<br/>Check accessibility & write test]
    end

    subgraph Worker[Phase 4: Worker Execution]
        K[Initialize Context<br/>Counters & Channels]
        L[Start Single Processor<br/>Backup + Delete]
        M[Start Bounded Walkers<br/>Concurrent Scanning]
        
        M --> N[Scan Paths<br/>Find Old Files]
        N --> O[Enqueue FileJobs<br/>With Per-Path Backup Flag]
        O --> P{Queue Full}
        P -->|Yes| O
        P -->|No| Q[Process Jobs Serially]
        
        Q --> R{Backup Enabled<br/>For This Job?}
        R -->|Yes| S[Copy to Backup]<br/>Check path accessible]
        R -->|No| T[Skip Backup]
        
        S --> U[Delete Source File]
        T --> U
        
        U --> V[Update Counters]
        V --> W[Cleanup Empty Dirs]
        W --> Q
        
        Q --> X{Stop Conditions}
        X -->|No| Q
        X -->|Yes| Y[Processor Exits]
    end

    subgraph Cleanup[Phase 5: Cleanup & Exit]
        Z[Close Jobs Channel]
        AA[Drain Remaining Jobs]
        AB[Log Per-Path Totals]
        AC[Log Summary Stats]
        AD[Exit with Code]
    end

    %% Startup Flow
    A --> B
    B --> C
    C --> D
    
    %% Setup Check Flow
    D --> E
    E -->|No| F
    F --> G
    E -->|Yes| H
    G --> AD
    
    %% Configuration Flow
    H --> I
    I -->|No| K
    I -->|Yes| J
    J -->|Fail| AD
    J -->|OK| K
    
    %% Worker Flow
    K --> L
    K --> M
    
    M -.-> |Discovered Files| O
    L -.-> |Processes| Q
    
    %% Shutdown Flow
    M --> |Walkers Done| Z
    Z --> AA
    AA --> Y
    Y --> AB
    AB --> AC
    AC --> AD

    %% Styling
    style Startup fill:#e1f5fe
    style SetupCheck fill:#fff3e0
    style Configuration fill:#e8f5e9
    style Worker fill:#fce4ec
    style Cleanup fill:#e1f5fe
```