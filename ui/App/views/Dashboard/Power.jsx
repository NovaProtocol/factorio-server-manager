import React, {useCallback, useEffect, useMemo, useState} from "react";
import Panel from "../../components/Panel";
import companion from "../../../api/resources/companion";
import SearchBox from "./SearchBox";
import Stat from "./Stat";
import TrendChart from "./TrendChart";
import WindowTabs from "./WindowTabs";
import {useScopedReport} from "./useScopedReport";
import {formatEnergy, formatPercent, formatPower, reportWatts} from "./helpers";

// Every planet's grid at once, so the page opens with the whole picture instead of
// an empty panel waiting for a surface to be chosen. A row selects the detail below.
const Planets = ({planets, selected, onSelect, report}) => (
    <table className="w-full text-left mb-4">
        <thead>
        <tr className="text-dirty-white">
            <th className="py-2">Planet</th>
            <th className="text-right">Networks</th>
            <th className="text-right">Production</th>
            <th className="text-right">Consumption</th>
            <th className="text-right">Demand met</th>
        </tr>
        </thead>
        <tbody>
        {planets.map(planet => (
            <tr key={planet.name}
                onClick={() => onSelect(planet.name)}
                className={`border-b border-gray-dark cursor-pointer ${planet.name === selected ? 'text-orange' : 'text-gray-light hover:text-dirty-white'}`}>
                <td className="py-2">{planet.name}</td>
                <td className="text-right">{planet.networks}</td>
                <td className="text-right">{formatPower(reportWatts(report, planet.production))}</td>
                <td className="text-right">{formatPower(reportWatts(report, planet.consumption))}</td>
                <td className={`text-right ${planet.consumption_satisfaction < 1 ? 'text-red-light' : ''}`}>
                    {formatPercent(planet.consumption_satisfaction)}
                </td>
            </tr>
        ))}
        </tbody>
    </table>
);

const PowerTable = ({title, entries, watts, series, onSelect}) => {
    const [filter, setFilter] = useState('');

    const shown = useMemo(() => {
        const needle = filter.trim().toLowerCase();
        return needle ? entries.filter(entry => entry.name.toLowerCase().includes(needle)) : entries;
    }, [entries, filter]);

    return (
        <Panel
            title={title}
            content={
                <>
                    <SearchBox value={filter} onChange={setFilter} shown={shown.length} total={entries.length}/>
                    <div className="max-h-96 overflow-y-auto fsm-scroll">
                        <table className="w-full text-left">
                            <thead>
                            <tr className="text-dirty-white">
                                <th className="py-2">Prototype</th>
                                <th className="text-right">Power</th>
                            </tr>
                            </thead>
                            <tbody>
                            {shown.map(entry => (
                                <tr key={entry.name}
                                    onClick={() => onSelect(entry.name)}
                                    className={`border-b border-gray-dark cursor-pointer ${series === entry.name ? 'text-orange' : 'text-gray-light hover:text-dirty-white'}`}>
                                    <td className="py-1">{entry.name}</td>
                                    <td className="py-1 text-right">{formatPower(watts(entry.value))}</td>
                                </tr>
                            ))}
                            </tbody>
                        </table>
                    </div>
                </>
            }
        />
    );
};

const Power = () => {

    const [surface, setSurface] = useState("");
    const [timeWindow, setTimeWindow] = useState("1h");
    const [series, setSeries] = useState("all");
    const [now, setNow] = useState(Date.now());

    useEffect(() => {
        const timer = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(timer);
    }, []);

    const fetchPower = useCallback(() => companion.power(surface, timeWindow), [surface, timeWindow]);
    const {payload, isLoading, updatedAt} = useScopedReport(`power|${surface}|${timeWindow}`, fetchPower);

    const report = payload?.report;
    const lifetime = timeWindow === 'all';

    // The server resolves a default surface when none is named, so the first
    // request works before any planet is known. The resolved name is shown but
    // never written back into state: that would change the key and fetch twice.
    const activeSurface = surface || report?.surface || '';

    const consumers = report?.consumers || [];
    const generators = report?.generators || [];
    const fetchGraph = useCallback(
        () => companion.graph(surface, 'power', series, timeWindow),
        [surface, series, timeWindow]);
    const {payload: graphPayload} = useScopedReport(
        `graph|${surface}|power|${series}|${timeWindow}`, fetchGraph);

    const graph = graphPayload?.report;
    const watts = value => reportWatts(report, value);
    const graphTitle = series === 'all' ? `${activeSurface} grid total` : series;
    const age = updatedAt ? Math.max(0, Math.round((now - updatedAt) / 1000)) : null;

    return (
        <>
            <Panel
                title="Power"
                className="mb-4"
                content={
                    <>
                        <div className="text-right text-sm mb-2 text-gray-light">
                            {age === null ? 'reading…' : `updated ${age}s ago · refreshes itself every 5s`}
                        </div>

                        {isLoading && !report
                            ? <p className="text-gray-light">Reading the grids…</p>
                            : payload?.error
                                ? <p className="text-red-light">{payload.error}</p>
                                : <Planets planets={report?.planets || []} selected={activeSurface}
                                           onSelect={setSurface} report={report}/>}

                        <WindowTabs value={timeWindow} onChange={setTimeWindow}/>

                        {report &&
                            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4 mt-4">
                                <Stat label="Production" value={formatPower(watts(report.production))}
                                      hint={lifetime ? 'lifetime average' : ''}/>
                                <Stat label="Consumption" value={formatPower(watts(report.consumption))}
                                      hint={lifetime ? 'lifetime average' : ''}/>
                                <Stat label="Capacity" value={formatPower(watts(report.max_production))} hint="this tick"/>
                                {/* Only demand met is a health figure. The game's production
                                    "satisfaction" is capacity utilisation — production over maximum
                                    production — so a low number means an idle grid, not a short one. */}
                                <Stat label="Demand met" value={formatPercent(report.consumption_satisfaction)}
                                      tone={report.consumption_satisfaction < 1 ? 'text-red-light' : 'text-green'}
                                      hint={report.consumption_satisfaction < 1 ? 'a grid is short' : 'demand covered'}/>
                                <Stat label="Accumulators"
                                      value={formatEnergy(report.accumulator_energy)}
                                      hint={`of ${formatEnergy(report.accumulator_capacity)}`}/>
                            </div>
                        }
                    </>
                }
            />

            <Panel
                title={`Graph — ${graphTitle}`}
                className="mb-4"
                actions={series !== 'all'
                    ? <button type="button" onClick={() => setSeries('all')}
                              className="accentuated bg-gray-light px-2 py-1 text-black font-bold hover:bg-orange">
                        Show the whole grid
                    </button>
                    : null}
                content={<TrendChart production={graph?.production} consumption={graph?.consumption}
                                      formatValue={formatPower}
                                      labels={{production: 'Generated', consumption: 'Drawn'}}/>}
            />

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
                <PowerTable title={`Consumers — ${report?.networks ?? '—'} networks`} entries={consumers.slice(0, 15)}
                            watts={watts} series={series} onSelect={setSeries}/>
                <PowerTable title="Generators" entries={generators} watts={watts} series={series} onSelect={setSeries}/>
            </div>

        </>
    );
};

export default Power;
