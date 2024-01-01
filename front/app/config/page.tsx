"use client";

import {useState} from "react";
import {Button, TextField} from "@mui/material";
import {Process, Token} from "@/app/lib/definitions";
import {DataGrid, GridColDef} from "@mui/x-data-grid";
import {enrichWithID} from "@/app/lib/utils";

function basicColumn(name: string, width: number = 150): GridColDef {
    return {field: name, headerName: name, width: width};
}

function humanize(num: number): string {
    let unit = "B";
    const gi = 1024 * 1024 * 1024;
    const mi = 1024 * 1024;
    const ki = 1024;
    if (num >= gi) {
        num /= gi;
        unit = "GiB";
    } else if (num >= mi) {
        num /= mi;
        unit = "MiB";
    } else if (num >= ki) {
        num /= ki;
        unit = "KiB";
    }
    if (unit !== "B") {
        let s = num.toString();
        const index = s.lastIndexOf('.');
        if (index != -1) {
            s = s.substring(0, index);
        }
        return `${s} ${unit}`;
    }
    return `${num} ${unit}`;
}

const columns: GridColDef[] = [
    basicColumn("PID"),
    basicColumn("PPID"),
    {
        field: "RSS",
        headerName: "RSS",
        valueFormatter: params => {
            return humanize(params.value);
        }
    },
    {
        field: "PSS",
        headerName: "PSS",
        valueFormatter: params => {
            return humanize(params.value);
        }
    },
    basicColumn("Path", 300),
    basicColumn("Args", 600)
];

export default function Page() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [message, setMessage] = useState("default message");
    const [token, setToken] = useState({} as Token);
    const [processes, setProcesses] = useState<Process[]>([]);

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
        const response = await fetch(`${host}/v1/processes`, {headers: {"Token": token.ID}});
        if (response.ok) {
            let items: Process[] = await response.json();
            setProcesses(enrichWithID(items.filter(v => v.PSS !== 0)));
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
            <p>{message}</p>
            <DataGrid columns={columns} rows={processes}/>
        </>
    );
}