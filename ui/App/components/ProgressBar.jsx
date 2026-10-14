import React from "react";

/**
 * A themed progress bar.
 *
 * Deliberately generic: give it a percentage and it renders, so any long running
 * action can show progress the same way.
 *
 * @param value      0 to 100
 * @param label      left hand text, e.g. "Uploading save"
 * @param detail     right hand text, defaults to the percentage
 * @param className  spacing from the caller
 */
const ProgressBar = ({value = 0, label, detail, className = ""}) => {
    const percent = Math.max(0, Math.min(100, Math.round(value)));

    return (
        <div className={className}>
            {(label || detail) &&
                <div className="flex justify-between text-sm mb-1">
                    <span className="text-dirty-white">{label}</span>
                    <span className="text-gray-light">{detail !== undefined ? detail : `${percent}%`}</span>
                </div>
            }
            <div className="w-full h-2 rounded-sm bg-gray-dark shadow-inner overflow-hidden">
                <div
                    className="h-full bg-orange transition-all duration-200 ease-out"
                    style={{width: `${percent}%`}}
                />
            </div>
        </div>
    );
}

export default ProgressBar;
