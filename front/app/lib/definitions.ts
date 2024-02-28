// Actually those field name in type are JSON tags.
// Not JavaScript style lowerCamelCase as Go default behaves that,
// so I don't need to explicit name the tags there.
// I accept the behaviour through the idea that web is case-insensitive.
// And as the column names in UI are better in UpperCamelCase.
// If I cast them before, would need to cast it back in the end.
// Some day a more comprehensive class style is introduced, better to turns it.

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

export type Exec = {
    WorkingDirectory: string
    Path: string
    Args: string[]
    RedirectPath: string
}

export type Application = {
    ID: number
    Name: string
    Exec: Exec
}

export type Node = {
    Process: Process
    Children: Node[]
}

function stat(root: Node): {
    processes: number
    memory: number
} {
    let ret = {
        processes: 1,
        memory: root.Process.PSS,
    }
    if (root.Children) {
        for (let child of root.Children) {
            let cs = stat(child);
            ret.processes += cs.processes;
            ret.memory += cs.memory;
        }
    }
    return ret
}

export type ApplicationComplex = Application & {
    Instances: Node[]
}

export type ApplicationLine = {
    id: number

    appId: number
    appName: string
    workingDirectory: string
    args: string[]
    redirectPath: string

    pid: number
    ppid: number
    processes: number
    memory: number // PSS sum
}

export function digestApplications(apps: ApplicationComplex[]): ApplicationLine[] {
    let ret: ApplicationLine[] = [];
    for (let app of apps) {
        const basic = {
            appId: app.ID,
            appName: app.Name,
            workingDirectory: app.Exec.WorkingDirectory,
            args: [app.Exec.Path, ...app.Exec.Args],
            redirectPath: app.Exec.RedirectPath,
        };
        if (app.Instances) {
            for (let root of app.Instances) {
                console.log(root);
                const line: ApplicationLine = {
                    id: ret.length,
                    ...basic,
                    pid: root.Process.PID,
                    ppid: root.Process.PPID,
                    ...stat(root)
                };
                ret.push(line);
            }
        } else {
            ret.push({
                id: ret.length,
                ...basic,
                pid: 0,
                ppid: 0,
                processes: 0,
                memory: 0
            });
        }
    }
    return ret
}