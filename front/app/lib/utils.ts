import {Process} from "@/app/lib/definitions";
import {GridColDef} from "@mui/x-data-grid";

export function enrichWithID(items: Process[]): (Process & { id: number })[] {
    return items.map(v => {
        return {
            id: v.PID,
            ...v
        };
    });
}

export function basicColumn(name: string, width: number = 150): GridColDef {
    return {field: name, headerName: name, width: width};
}
