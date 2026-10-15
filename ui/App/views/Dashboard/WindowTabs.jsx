import React from "react";
import {WINDOWS} from "./helpers";

// The time-window picker, built the way the settings pages build their tabs. The
// tabs share the row equally rather than sizing to their labels, so the row fills
// the panel and the nine of them are all the same width.
const WindowTabs = ({value, onChange}) => (
    <div className="bg-gray-dark px-4 pt-3 mb-4 rounded-t-sm flex">
        {WINDOWS.map(option => (
            <span
                key={option.value}
                title={option.title}
                onClick={() => onChange(option.value)}
                className={
                    'flex-1 text-center accentuated-t accentuated-x cursor-pointer rounded-t py-1 font-bold '
                    + (option.value === value
                        ? 'text-dirty-white bg-gray-dark'
                        : 'text-black bg-gray-light hover:bg-orange')
                }
            >
                {option.name}
            </span>
        ))}
    </div>
);

export default WindowTabs;
