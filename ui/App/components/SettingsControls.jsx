import React, {useEffect, useRef, useState} from "react";

/**
 * The controls both settings pages share, shaped like Factorio's own settings
 * screen: a label, one control, and a question mark at the end that shows the
 * mod's description. Booleans are checkboxes, numbers take numbers only, a value
 * with a fixed set is a dropdown, and anything else is a text box.
 */

const INPUT = 'shadow appearance-none border w-full py-2 px-3 text-black';
const CARET = 'pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-black';

// The lighter, smaller notes: "(default)", "min 1". They annotate a value rather
// than being one.
export const Note = ({children, className = ""}) => (
    <span className={`text-sm font-normal text-gray-light ${className}`}>{children}</span>
);

const Caret = () => (
    <div className={CARET}>
        <svg className="fill-current h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
            <path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z"/>
        </svg>
    </div>
);

/**
 * Options are [{value, label, isDefault}]. The value that is the default carries
 * a light "(default)" beside it, so the default reads as an annotation on the
 * real value rather than as a second entry of its own.
 */
export const Dropdown = ({value, options, disabled = false, onChange}) => {
    const [isOpen, setIsOpen] = useState(false);
    const container = useRef(null);

    useEffect(() => {
        if (!isOpen) {
            return;
        }

        const close = event => {
            if (container.current && !container.current.contains(event.target)) {
                setIsOpen(false);
            }
        };

        document.addEventListener('mousedown', close);
        return () => document.removeEventListener('mousedown', close);
    }, [isOpen]);

    const selected = options.find(option => String(option.value) === String(value));

    return (
        <div className="relative" ref={container}>
            <button type="button"
                    disabled={disabled}
                    onClick={() => setIsOpen(open => !open)}
                    className={`${INPUT} relative text-left ${disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}`}>
                {selected ? selected.label : value}
                {selected && selected.isDefault &&
                    <Note className="ml-2">(default)</Note>
                }
                <Caret/>
            </button>
            {isOpen &&
                <div className="absolute z-30 mt-1 w-full bg-white shadow-lg">
                    {options.map(option => (
                        <div key={String(option.value)}
                             onClick={() => {
                                 setIsOpen(false);
                                 onChange(option.value);
                             }}
                             className="cursor-pointer px-3 py-2 text-black hover:bg-gray-light">
                            {option.label}
                            {option.isDefault &&
                                <Note className="ml-2">(default)</Note>
                            }
                        </div>
                    ))}
                </div>
            }
        </div>
    );
};

// Numbers take numbers. The browser enforces it for typed input; this also drops
// anything else that arrives, such as pasted text.
export const NumberField = ({value, placeholder, disabled = false, onChange}) => (
    <input className={INPUT}
           type="number"
           inputMode="decimal"
           value={value}
           placeholder={placeholder}
           disabled={disabled}
           onChange={event => onChange(event.target.value.replace(/[^0-9.eE+-]/g, ''))}/>
);

export const TextField = ({value, placeholder, disabled = false, onChange}) => (
    <input className={INPUT}
           type="text"
           value={value}
           placeholder={placeholder}
           disabled={disabled}
           onChange={event => onChange(event.target.value)}/>
);

export const CheckboxField = ({checked, disabled = false, onChange}) => (
    <input className="leading-tight h-4 w-4"
           type="checkbox"
           checked={checked}
           disabled={disabled}
           onChange={event => onChange(event.target.checked)}/>
);

/**
 * The description, behind a question mark next to the setting's name. The message
 * is drawn here rather than left to the browser's own tooltip, which is slow and
 * easy to miss.
 */
export const QuestionMark = ({description}) => {
    const [isOpen, setIsOpen] = useState(false);

    if (!description) {
        return null;
    }

    return (
        <span className="relative inline-block ml-2 align-middle">
            <span
                onMouseEnter={() => setIsOpen(true)}
                onMouseLeave={() => setIsOpen(false)}
                className="cursor-help text-gray-light hover:text-orange"
            >
                ?
            </span>
            {isOpen &&
                <span className="absolute left-0 top-5 z-40 block w-72 rounded-sm bg-black bg-opacity-90 px-3 py-2 text-sm font-normal text-white shadow-lg">
                    {description}
                </span>
            }
        </span>
    );
};

/**
 * One setting: its label and any annotation on the left, the control in the
 * middle, the description behind a question mark at the end.
 */
export const SettingRow = ({label, note = null, description = null, actions = null, children}) => (
    <div className="flex flex-wrap items-start gap-x-4 py-2">
        <div className="min-w-0 grow basis-2/5">
            {label}
            <QuestionMark description={description}/>
            {note && <Note className="ml-2">{note}</Note>}
        </div>
        <div className="w-1/2 shrink-0">{children}</div>
        <div className="flex shrink-0 items-center gap-x-3 pt-2">{actions}</div>
    </div>
);

// A heading for a group of settings, used to separate one mod from the next.
export const SettingGroup = ({title, note = null}) => (
    <div className="mt-5 border-t border-gray-light pt-3 font-bold text-dirty-white">
        {title}
        {note && <Note className="ml-3">{note}</Note>}
    </div>
);
