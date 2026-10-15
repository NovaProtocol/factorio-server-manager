import React from "react";

const Stat = ({label, value, hint, tone = 'text-dirty-white'}) => (
    <div className="bg-black rounded-sm shadow-inner p-4 text-center">
        <div className="text-gray-light text-sm">{label}</div>
        <div className={`text-2xl font-bold ${tone}`}>{value}</div>
        {hint && <div className="text-gray-light text-sm">{hint}</div>}
    </div>
);

export default Stat;
