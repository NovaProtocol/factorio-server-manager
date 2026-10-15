import React from "react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faSearch, faTimes} from "@fortawesome/free-solid-svg-icons";

// A filter for one panel's table. Kept on the dark panel background rather than the
// white form styling the settings pages use, since it sits inside a data panel.
const SearchBox = ({value, onChange, placeholder = 'Filter by name', shown, total}) => (
    <div className="flex items-center bg-black shadow-inner px-2 mb-2">
        <FontAwesomeIcon className="text-gray-light mr-2" icon={faSearch}/>
        <input
            type="text"
            className="bg-transparent text-dirty-white py-1 w-full outline-none placeholder-gray-light"
            placeholder={placeholder}
            value={value}
            onChange={event => onChange(event.target.value)}
        />
        {value &&
            <>
                <span className="text-gray-light text-sm mr-2 whitespace-nowrap">{shown} / {total}</span>
                <FontAwesomeIcon
                    className="text-gray-light cursor-pointer hover:text-orange"
                    title="Clear the filter"
                    icon={faTimes}
                    onClick={() => onChange('')}
                />
            </>
        }
    </div>
);

export default SearchBox;
