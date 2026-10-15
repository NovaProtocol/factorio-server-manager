import React, {useEffect, useState} from "react";
import Dropdown from "../../components/Dropdown";

// The controls the dashboard pages share. The time window is not here: it is a row
// of tabs (WindowTabs) rather than a dropdown. A page that has its own table for
// picking a surface (Power) leaves the dropdown off entirely.
const Controls = ({surfaces, surface, onSurface, updatedAt, dataAge, hideSurface = false, children}) => {

    // The manager answers from its own cache, so the browser's clock is no longer
    // the interesting one: it says when a response arrived, not how old the numbers
    // are. The payload's own age is the honest figure, and it ticks up here between
    // polls so a reader sees it grow.
    const [now, setNow] = useState(Date.now());

    useEffect(() => {
        const timer = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(timer);
    }, []);

    const sinceReceived = updatedAt ? Math.max(0, Math.round((now - updatedAt) / 1000)) : 0;
    const age = dataAge === null || dataAge === undefined
        ? (updatedAt ? sinceReceived : null)
        : dataAge + sinceReceived;
    // Five minutes is where the manager calls a report stale (api staleAfter).
    const stale = age !== null && age > 300;

    // The manager keeps its cache in memory and answers from it however old it is,
    // so the age can
    // be minutes or hours old rather than seconds — say which.
    const ageText = seconds => {
        if (seconds < 60) {
            return `${seconds}s`;
        }
        if (seconds < 3600) {
            return `${Math.floor(seconds / 60)}m`;
        }
        if (seconds < 86400) {
            return `${Math.floor(seconds / 3600)}h`;
        }
        return `${Math.floor(seconds / 86400)}d`;
    };

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
                {age === null ? 'reading…' : `updated ${ageText(age)} ago`}
                {stale ? ' (stale)' : ''}
                <span className="ml-2">· refreshes itself every 5s</span>
            </div>
        </div>
    );
};

export default Controls;
