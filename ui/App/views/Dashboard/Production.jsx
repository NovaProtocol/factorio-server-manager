import React, {useCallback, useMemo, useState} from "react";
import Panel from "../../components/Panel";
import companion from "../../../api/resources/companion";
import Controls from "./Controls";
import SearchBox from "./SearchBox";
import TrendChart from "./TrendChart";
import WindowTabs from "./WindowTabs";
import {useScopedReport} from "./useScopedReport";
import {formatNumber, formatRate, windowName} from "./helpers";

const bySize = (a, b) => (b.rate ?? b.count ?? 0) - (a.rate ?? a.count ?? 0);

// Every planet's output at once, so the page opens with the whole picture rather
// than one surface picked from a dropdown. A row selects the detail below.
const Planets = ({planets, selected, onSelect, lifetime, window}) => (
    <table className="w-full text-left mb-4">
        <thead>
        <tr className="text-dirty-white">
            <th className="py-2">Planet</th>
            <th className="text-right">Item kinds</th>
            <th className="text-right whitespace-nowrap">{lifetime ? 'Items /min' : `Items /min (${windowName(window)})`}</th>
            <th className="text-right whitespace-nowrap">All-time items</th>
            <th className="text-right">Built</th>
        </tr>
        </thead>
        <tbody>
        {planets.map(planet => (
            <tr key={planet.name}
                onClick={() => onSelect(planet.name)}
                className={`border-b border-gray-dark cursor-pointer ${planet.name === selected ? 'text-orange' : 'text-gray-light hover:text-dirty-white'}`}>
                <td className="py-2">{planet.name}</td>
                <td className="text-right">{formatNumber(planet.item_kinds)}</td>
                <td className="text-right">{lifetime ? 'n/a' : formatRate(planet.items_rate)}</td>
                <td className="text-right">{formatNumber(planet.items)}</td>
                <td className="text-right">{formatNumber(planet.builds)}</td>
            </tr>
        ))}
        </tbody>
    </table>
);

// One list of items or fluids. A row is the graph picker: clicking it plots that
// series, which is quicker than hunting through a dropdown of 199 entries.
const FlowTable = ({title, entries, window, lifetime, selected, onSelect}) => {
    const [filter, setFilter] = useState('');
    const sorted = useMemo(() => [...entries].sort(bySize), [entries]);

    const shown = useMemo(() => {
        const needle = filter.trim().toLowerCase();
        return needle ? sorted.filter(entry => entry.name.toLowerCase().includes(needle)) : sorted;
    }, [sorted, filter]);

    return (
        <Panel
            title={`${title} — ${entries.length}`}
            content={
                <>
                    <SearchBox value={filter} onChange={setFilter} shown={shown.length} total={entries.length}/>
                    <div className="max-h-96 overflow-y-auto fsm-scroll">
                        <table className="w-full text-left">
                            <thead>
                            <tr className="text-dirty-white">
                                <th className="py-2">Name</th>
                                <th className="text-right whitespace-nowrap">{lifetime ? 'Total' : `Per minute (${windowName(window)})`}</th>
                                {!lifetime && <th className="text-right whitespace-nowrap">In window</th>}
                            </tr>
                            </thead>
                            <tbody>
                            {shown.map(entry => (
                                <tr key={entry.name}
                                    onClick={() => onSelect(entry.name)}
                                    className={`border-b border-gray-dark cursor-pointer ${selected === entry.name ? 'text-orange' : 'text-gray-light hover:text-dirty-white'}`}>
                                    <td className="py-1">{entry.name}</td>
                                    <td className="py-1 text-right">
                                        {lifetime ? formatNumber(entry.count) : formatRate(entry.rate)}
                                    </td>
                                    {!lifetime &&
                                        <td className="py-1 text-right">{formatNumber(entry.count)}</td>
                                    }
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

const Production = () => {

    const [surface, setSurface] = useState("");
    const [timeWindow, setTimeWindow] = useState("1h");
    const [selected, setSelected] = useState(null);
    const fetchProduction = useCallback(() => companion.production(surface, timeWindow), [surface, timeWindow]);
    const {payload, isLoading, updatedAt} = useScopedReport(`production|${surface}|${timeWindow}`, fetchProduction);

    const report = payload?.report;
    const activeSurface = surface || report?.surface || '';
    const lifetime = timeWindow === 'all';

    const items = report?.items || [];
    const fluids = report?.fluids || [];

    // Default to the biggest item so the graph opens on something with a shape.
    const biggest = useMemo(() => [...items].sort(bySize)[0], [items]);
    const chosen = selected ?? biggest?.name ?? null;
    const isFluid = fluids.some(entry => entry.name === chosen);
    const category = isFluid ? 'fluid' : 'item';

    const fetchGraph = useCallback(() => {
        if (!chosen) {
            return Promise.resolve(null);
        }
        return companion.graph(surface, category, chosen, timeWindow);
    }, [surface, category, chosen, timeWindow]);
    const {payload: graphPayload} = useScopedReport(
        `graph|${surface}|${category}|${chosen}|${timeWindow}`, fetchGraph);

    const graph = graphPayload?.report;

    return (
        <>
            <Panel
                title="Production"
                className="mb-4"
                content={
                    <>
                        <Controls surfaces={[]} surface={activeSurface} onSurface={setSurface}
                                  updatedAt={updatedAt} hideSurface/>

                        {isLoading && !report
                            ? <p className="text-gray-light">Reading production…</p>
                            : payload?.error
                                ? <p className="text-red-light">{payload.error}</p>
                                : <Planets planets={report?.planets || []} selected={activeSurface}
                                           onSelect={setSurface} lifetime={lifetime} window={timeWindow}/>}

                        <WindowTabs value={timeWindow} onChange={setTimeWindow}/>
                    </>
                }
            />

            {chosen &&
                <Panel
                    title={`Graph — ${chosen} per minute (${lifetime ? '1000h precision' : windowName(timeWindow)})`}
                    className="mb-4"
                    content={<TrendChart production={graph?.production} consumption={graph?.consumption}
                                          formatValue={formatRate}
                                          labels={{production: 'Produced', consumption: 'Consumed'}}/>}
                />
            }

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
                <FlowTable title="Items produced" entries={items} window={timeWindow} lifetime={lifetime}
                           selected={chosen} onSelect={name => setSelected(name)}/>
                <FlowTable title="Fluids produced" entries={fluids} window={timeWindow} lifetime={lifetime}
                           selected={chosen} onSelect={name => setSelected(name)}/>
            </div>
        </>
    );
};

export default Production;
