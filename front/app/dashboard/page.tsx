// noinspection DuplicatedCode
// I would implement it first, then extract the same things in config page properly.
// The core question is how shall I store the common worker info of username & password.
"use client";

import {useState} from "react";
import {Button, FormControlLabel, FormGroup, Switch, TextField} from "@mui/material";
import RefreshIcon from '@mui/icons-material/Refresh';
import {ApplicationComplex, ApplicationLine, digestApplications, Token} from "@/app/lib/definitions";
import {DataGrid, GridColDef, GridRenderCellParams} from "@mui/x-data-grid";
import {humanize} from "@/app/lib/humanize";

const columns = (runFunc: (appId: number) => void,
                 killFunc: (pid: number) => void,
                 pendingAppIds: Set<number>,
                 pendingProcessIds: Set<Number>): GridColDef[] => [
    {
        field: "stoppable",
        headerName: "Action",
        width: 100,
        valueGetter: (params): boolean => {
            return params.row.memory > 0 && params.row.pid > 0;
        },
        renderCell: (params: GridRenderCellParams<ApplicationLine, boolean>) => {
            if (params.value) {
                if (pendingProcessIds.has(params.row.pid)) {
                    return <Button size="small" variant="contained" color="error"><RefreshIcon/></Button>;
                } else {
                    return <Button size="small" variant="contained" color="error"
                                   onClick={() => {
                                       killFunc(params.row.pid);
                                   }}>Kill</Button>;
                }
            } else {
                if (pendingAppIds.has(params.row.appId)) {
                    return <Button size="small" variant="contained" color="success"><RefreshIcon/></Button>;
                } else {
                    return <Button size="small" variant="contained" color="success"
                                   onClick={() => {
                                       runFunc(params.row.appId);
                                   }}>Run</Button>;
                }
            }
        },
    },
    {field: "pid", headerName: "PID"},
    {field: "ppid", headerName: "PPID"},
    {
        field: "memory",
        headerName: "Memory",
        valueFormatter: params => {
            return humanize(params.value);
        }
    },
    {field: "processes", headerName: "Processes"},

    {field: "appId", headerName: "AppID"},
    {field: "appName", headerName: "AppName"},
    {field: "workingDirectory", headerName: "PWD", width: 225}, // ENV $PWD
    {field: "args", headerName: "args", width: 450},
    {field: "redirectPath", headerName: ">", width: 225}
];

export default function Page() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [token, setToken] = useState({} as Token);
    const [applications, setApplications] = useState<ApplicationLine[]>([]);
    const [updateAfterAction, setUpdateAfterAction] = useState(true);
    const [pendingProcessIds, setPendingProcessIds] = useState<number[]>([]);
    const [pendingApplicationIds, setPendingApplicationIds] = useState<number[]>([]);
    const host = "https://hyisen.net"
    const updateAfterActionDelayMs: number = 500; // In practice, kill needs time to finish after DELETE is return.
    const pendingBlockingExtensionMs: number = 500; // A bit more to prevent flash by simultaneous unblock and update.

    async function login() {
        const response = await fetch(`${host}/v1/session`, {
            method: "POST",
            mode: "cors",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({username: username, password: password})
        });
        if (response.ok) {
            setToken(await response.json());
        }
    }

    async function fetchData() {
        const response = await fetch(`${host}/v1/applications`, {headers: {"Token": token.ID}});
        if (response.ok) {
            let items: ApplicationComplex[] = await response.json();
            setApplications(digestApplications(items));
        }
    }

    async function runApplication(appId: number) {
        if (updateAfterAction) {
            setPendingApplicationIds([...pendingApplicationIds, appId]);
        }
        const response = await fetch(`${host}/v1/applications/${appId}/instances`,
            {method: "PUT", headers: {"Token": token.ID}});
        if (!response.ok) {
            setPendingApplicationIds(pendingApplicationIds.filter(v => v != appId));
            alert(`bad code ${response.status}: ${await response.text()}`);
        } else if (updateAfterAction) {
            setTimeout(() => {
                fetchData();
                setTimeout(() => {
                    setPendingApplicationIds(pendingApplicationIds.filter(v => v != appId));
                }, pendingBlockingExtensionMs)
            }, updateAfterActionDelayMs);
        }
    }

    async function killProcess(pid: number) {
        if (updateAfterAction) {
            setPendingProcessIds([...pendingProcessIds, pid]);
        }
        const response = await fetch(`${host}/v1/processes/${pid}`,
            {method: "DELETE", headers: {"Token": token.ID}});
        if (!response.ok) {
            setPendingProcessIds(pendingProcessIds.filter(v => v != pid));
            alert(`bad code ${response.status}: ${await response.text()}`);
        } else if (updateAfterAction) {
            setTimeout(() => {
                fetchData();
                setTimeout(() => {
                    setPendingProcessIds(pendingProcessIds.filter(v => v != pid));
                }, pendingBlockingExtensionMs)
            }, updateAfterActionDelayMs);
        }
    }

    return (
        <>
            <Button variant="contained" onClick={() => {
                login().then();
            }}>Login</Button>
            <Button variant="contained" onClick={() => {
                fetchData().then();
            }}>Fetch</Button>
            <TextField label="username"
                       value={username}
                       onChange={event => setUsername(event.target.value)}></TextField>
            <TextField label="password"
                       type="password"
                       value={password}
                       onChange={event => setPassword(event.target.value)}></TextField>
            <FormGroup>
                <FormControlLabel control={<Switch checked={updateAfterAction}
                                                   onChange={event => setUpdateAfterAction(event.target.checked)}/>}
                                  label={"UpdateAfterAction"}/>
            </FormGroup>
            <br/>
            <DataGrid columns={columns(appId => {
                runApplication(appId).then();
            }, pid => {
                killProcess(pid).then();
            }, new Set<number>(pendingApplicationIds), new Set<number>(pendingProcessIds))} rows={applications}/>
        </>
    );
}