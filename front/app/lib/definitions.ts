export type Token = {
    ID: string;
    ExpireAt: Date;
    Username: string;
}

export type Process = {
    Path: string;
    Args: string[];
    PID: number;
    PPID: number;
    RSS: number;
    PSS: number;
}