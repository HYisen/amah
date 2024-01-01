import {Process} from "@/app/lib/definitions";

export function enrichWithID(items: Process[]): (Process & { id: number })[] {
    return items.map(v => {
        return {
            id: v.PID,
            ...v
        };
    });
}