"use client";

import {useState} from "react";
import {Button, TextField} from "@mui/material";

export default function Page() {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [message, setMessage] = useState("default message");

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
        console.log(response.status);
        setMessage(JSON.stringify(await response.json()));
    }

    return (
        <>
            <Button variant="contained" onClick={() => {
                login().then(() => window.console.log("login done"));
            }}>FIRE</Button>
            <TextField label="username"
                       value={username}
                       onChange={event => setUsername(event.target.value)}></TextField>
            <TextField label="password"
                       type="password"
                       value={password}
                       onChange={event => setPassword(event.target.value)}></TextField>
            <br/>
            <p>{message}</p>
        </>
    )
}
;