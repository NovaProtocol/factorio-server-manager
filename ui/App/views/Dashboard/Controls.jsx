import React, {useEffect, useState} from "react";
import Dropdown from "../../components/Dropdown";

// The controls the dashboard pages share. The time window is not here: it is a row
// of tabs (WindowTabs) rather than a dropdown. A page that has its own table for
// picking a surface (Power) leaves the dropdown off entirely.
const Controls = ({surfaces, surface, onSurface, updatedAt, hideSurface = false, children}) => {

    // The age is counted here, from when the browser received the payload. The age
    // the server reports is a snapshot of its own cache, so showing that verbatim
    // would freeze on "0s ago" and claim the page was updating when it was not.
    const [now, setNow] = useState(Date.now());

    useEffect(() => {
        const timer = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(timer);
    }, []);

    const age = updatedAt ? Math.max(0, Math.round((now - updatedAt) / 1000)) : null;
    const stale = age !== null && age > 45;

    return (
        <div className="flex flex-wrap items-center gap-4 mb-3">
            {!hideSurface &&
                <div className="w-44">
                    <span className="text-gray-light text-sm block mb-1">Surface</span>
                    <Dropdown value={surface} options={surfaces} onChange={onSurface} placeholder="default"/>
                </div>
            }
            {children}
            <div className={`flex-1 text-right text-sm ${stale ? 'text-red-light' : 'text-gray-light'}`}>
                {age === null ? 'reading…' : `updated ${age}s ago`}
                {stale ? ' (stale)' : ''}
                <span className="ml-2">· refreshes itself every 5s</span>
            </div>
        </div>
    );
};

export default Controls;
