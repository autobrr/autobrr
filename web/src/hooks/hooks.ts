import React from "react";

export function useToggle(initialValue: boolean = false): [boolean, () => void] {
    const [value, setValue] = React.useState<boolean>(initialValue);
    const toggle = React.useCallback(() => {
        setValue(v => !v);
    }, []);
    return [value, toggle];
}
