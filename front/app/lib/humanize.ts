export function humanize(num: number): string {
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