import React, {useEffect, useRef, useState} from "react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faChevronDown} from "@fortawesome/free-solid-svg-icons";

// A themed dropdown: a button and its own list, so it matches the panels instead
// of rendering the browser's native select popup.
const Dropdown = ({value, options, onChange, placeholder = '—', className = '', disabled = false}) => {

    const [isOpen, setIsOpen] = useState(false);
    const container = useRef(null);

    useEffect(() => {
        if (!isOpen) {
            return;
        }

        const onDocumentMouseDown = event => {
            if (container.current && !container.current.contains(event.target)) {
                setIsOpen(false);
            }
        };
        const onDocumentKeyDown = event => {
            if (event.key === 'Escape') {
                setIsOpen(false);
            }
        };

        document.addEventListener('mousedown', onDocumentMouseDown);
        document.addEventListener('keydown', onDocumentKeyDown);

        return () => {
            document.removeEventListener('mousedown', onDocumentMouseDown);
            document.removeEventListener('keydown', onDocumentKeyDown);
        };
    }, [isOpen]);

    const selected = options.find(option => option.value === value);

    return (
        <div className={`relative ${className}`} ref={container}>
            <button
                type="button"
                disabled={disabled}
                onClick={() => setIsOpen(open => !open)}
                className={
                    'accentuated bg-gray-light text-black font-bold py-2 px-3 w-full text-left flex items-center justify-between '
                    + (disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer hover:bg-orange')
                }
            >
                <span>{selected ? selected.name : placeholder}</span>
                <FontAwesomeIcon className="ml-3" icon={faChevronDown}/>
            </button>

            {isOpen &&
                <ul className="absolute z-30 mt-1 w-full max-h-96 overflow-y-auto accentuated bg-gray-dark shadow-xl">
                    {options.map(option => (
                        <li
                            key={option.value}
                            onClick={() => {
                                onChange(option.value);
                                setIsOpen(false);
                            }}
                            className={
                                'cursor-pointer px-3 py-2 '
                                + (option.value === value
                                    ? 'bg-orange text-black font-bold'
                                    : 'text-dirty-white hover:bg-gray-medium')
                            }
                        >
                            {option.name}
                        </li>
                    ))}
                </ul>
            }
        </div>
    );
};

export default Dropdown;
