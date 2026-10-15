import React from "react";

const Input = ({
                   register,
                   placeholder = undefined,
                   type = "text",
                   defaultValue = undefined,
                   hasAutoComplete = true,
                   onChange = undefined,
                   onBlur = undefined,
                   onKeyDown = () => undefined,
                   min = undefined,
                   max = undefined,
                   value = undefined,
                   disabled = false
               }) => {
    // react-hook-form hands its own onChange/onBlur in through `register`, and a
    // plain attribute after the spread would replace them — which silently broke
    // every form in the app, because the library then never saw what was typed.
    // Both are called instead: `register` first, then whatever the caller passed.
    const handleChange = event => {
        register?.onChange?.(event);
        onChange?.(event);
    };

    const handleBlur = event => {
        register?.onBlur?.(event);
        onBlur?.(event);
    };

    return (
        <input
            className="shadow appearance-none border w-full py-2 px-3 text-black"
            placeholder={placeholder}
            {...register}
            type={type}
            onChange={handleChange}
            onBlur={handleBlur}
            onKeyDown={onKeyDown}
            autoComplete={hasAutoComplete ? "on" : "off"}
            defaultValue={defaultValue}
            min={min}
            max={max}
            value={value}
            disabled={disabled}
        />
    )
}

export default Input;
