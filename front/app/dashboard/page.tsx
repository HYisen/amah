// noinspection DuplicatedCode
// I would implement it first, then extract the same things in config page properly.
// The core question is how shall I store the common worker info of username & password.
"use client";

import {useState} from "react";
import {Button, TextField} from "@mui/material";
import {ApplicationComplex, ApplicationLine, digestApplications, Token} from "@/app/lib/definitions";
import {DataGrid, GridColDef, GridRenderCellParams} from "@mui/x-data-grid";
import {humanize} from "@/app/lib/humanize";

const columns = (runFunc: (appId: number) => void, killFunc: (pid: number) => void): GridColDef[] => [
    {
        field: "stoppable",
        headerName: "Action",
        width: 100,
        valueGetter: (params): boolean => {
            return params.row.memory > 0 && params.row.pid > 0;
        },
        renderCell: (params: GridRenderCellParams<ApplicationLine, boolean>) => {
            return params.value ?
                <Button size="small" variant="contained" color="error"
                        onClick={() => {
                            killFunc(params.row.pid);
                        }}>Kill</Button>
                :
                <Button size="small" variant="contained" color="success"
                        onClick={() => {
                            runFunc(params.row.appId);
                        }}>Run</Button>
                ;
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

    const host = "https://hyisen.net"

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
        const response = await fetch(`${host}/v1/applications/${appId}/instances`,
            {method: "PUT", headers: {"Token": token.ID}});
        if (!response.ok) {
            alert(`bad code ${response.status}: ${await response.text()}`);
        }
    }

    async function killProcess(pid: number) {
        const response = await fetch(`${host}/v1/processes/${pid}`,
            {method: "DELETE", headers: {"Token": token.ID}});
        if (!response.ok) {
            alert(`bad code ${response.status}: ${await response.text()}`);
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
            <br/>
            <DataGrid columns={columns(appId => {
                runApplication(appId).then();
            }, pid => {
                killProcess(pid).then();
            })} rows={applications}/>
        </>
    );
}