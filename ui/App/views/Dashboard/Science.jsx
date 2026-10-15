import React, {useCallback, useEffect, useMemo, useState} from "react";
import Panel from "../../components/Panel";
import companion from "../../../api/resources/companion";
import Controls from "./Controls";
import SearchBox from "./SearchBox";
import Stat from "./Stat";
import TrendChart from "./TrendChart";
import WindowTabs from "./WindowTabs";
import {useScopedReport} from "./useScopedReport";
import {formatNumber, formatPercent, formatRate, windowName} from "./helpers";

// The mod measures the rate off the research bar and keeps it while the game is
// paused (a frozen game advances no ticks, but the last measurement stays the best
// available). Once that held reading is more than a game-time minute old, say so.
const MINUTE_TICKS = 60 * 60;
const effectiveHint = report => {
    if (!report?.productivity_ratio) return 'no reading yet — needs a running game';
    const age = report.research_rate_age_ticks ?? 0;
    const held = Math.round(age / 60 / 60 * 10) / 10;
    return age > MINUTE_TICKS
        ? `${report.productivity_ratio.toFixed(2)}x the pack rate — held, measured ${held} game-minutes ago`
        : `${report.productivity_ratio.toFixed(2)}x the pack rate, measured from the bar`;
};

const Science = () => {
    const [surface, setSurface] = useState("");
    const [timeWindow, setTimeWindow] = useState("1m");
    const [selected, setSelected] = useState("all");

    const fetchScience = useCallback(() => companion.science(surface, timeWindow), [surface, timeWindow]);
    const {payload, isLoading, updatedAt} = useScopedReport(`science|${surface}|${timeWindow}`, fetchScience);

    const report = payload?.report;
    const activeSurface = surface || report?.surface || '';
    const research = report?.research || {};
    const packs = report?.packs || [];
    const lifetime = timeWindow === 'all';

    const surfaces = (report?.surfaces || []).map(entry => ({value: entry.name, name: entry.name}));

    const fetchGraph = useCallback(
        () => companion.graph(surface, 'item', selected, timeWindow),
        [surface, selected, timeWindow]);
    const {payload: graphPayload} = useScopedReport(
        `graph|${surface}|item|${selected}|${timeWindow}`, fetchGraph);

    const graph = graphPayload?.report;
    const sorted = useMemo(
        () => [...packs].sort((a, b) => (b.consumed ?? b.produced ?? 0) - (a.consumed ?? a.produced ?? 0)),
        [packs]);

    const [filter, setFilter] = useState('');
    const shownPacks = useMemo(() => {
        const needle = filter.trim().toLowerCase();
        return needle ? sorted.filter(pack => pack.name.toLowerCase().includes(needle)) : sorted;
    }, [sorted, filter]);

    return (
        <>
            <Panel
                title="Science"
                className="mb-4"
                content={
                    <>
                        <Controls surfaces={surfaces} surface={activeSurface} onSurface={setSurface}
                                  updatedAt={updatedAt}/>

                        <WindowTabs value={timeWindow} onChange={setTimeWindow}/>

                        {isLoading && !report
                            ? <p className="text-gray-light">Reading the labs…</p>
                            : payload?.error
                                ? <p className="text-red-light">{payload.error}</p>
                                : null}

                        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4 mt-4">
                            <Stat
                                label="Science per minute"
                                value={lifetime ? '—' : formatNumber(report?.spm)}
                                hint={lifetime ? 'pick a window' : 'slowest pack this research needs'}
                            />
                            {/* The packs are what the labs eat; the research bar moves
                                faster once lab productivity is counted, so the measured
                                number is the real one. */}
                            <Stat
                                label="Effective research"
                                value={report?.research_rate ? formatNumber(report.research_rate) : '—'}
                                hint={effectiveHint(report)}
                                tone="text-orange"
                            />
                            <Stat label="Current research" value={research.current ?? '—'}
                                  hint={`${formatPercent(research.progress)} complete`}/>
                            <Stat label="Technologies"
                                  value={`${research.researched ?? '—'} / ${research.total ?? '—'}`}
                                  hint={`${research.queued ?? 0} queued`}/>
                            <Stat label="Rockets launched" value={formatNumber(research.rockets_launched)}/>
                        </div>
                    </>
                }
            />

            <Panel
                title={`Graph — ${selected === 'all' ? 'all packs consumed' : selected} (${lifetime ? '1000h precision' : windowName(timeWindow)})`}
                className="mb-4"
                content={<TrendChart production={graph?.production} consumption={graph?.consumption}
                                      formatValue={formatRate}
                                      labels={{production: 'Produced', consumption: 'Consumed'}}/>}
            />

            <Panel
                title="Packs"
                className="mb-6"
                content={
                    <>
                        <SearchBox value={filter} onChange={setFilter}
                                   shown={shownPacks.length} total={sorted.length}/>
                        <div className="max-h-96 overflow-y-auto fsm-scroll">
                            <table className="w-full text-left">
                                <thead>
                                <tr className="text-dirty-white">
                                    <th className="py-2">Pack</th>
                                    <th className="text-right whitespace-nowrap">Needed</th>
                                    <th className="text-right whitespace-nowrap">{lifetime ? 'Produced (total)' : 'Produced /min'}</th>
                                    <th className="text-right whitespace-nowrap">{lifetime ? 'Consumed (total)' : 'Consumed /min'}</th>
                                </tr>
                                </thead>
                                <tbody>
                                <tr onClick={() => setSelected('all')}
                                    className={`border-b border-gray-dark cursor-pointer ${selected === 'all' ? 'text-orange' : 'text-gray-light hover:text-dirty-white'}`}>
                                    <td className="py-1 font-bold">All packs</td>
                                    <td className="py-1 text-right">—</td>
                                    <td className="py-1 text-right">—</td>
                                    <td className="py-1 text-right">—</td>
                                </tr>
                                {shownPacks.map(pack => (
                                    <tr key={pack.name}
                                        onClick={() => setSelected(pack.name)}
                                        className={`border-b border-gray-dark cursor-pointer ${selected === pack.name ? 'text-orange' : 'text-gray-light hover:text-dirty-white'}`}>
                                        <td className="py-1">{pack.name}</td>
                                        <td className={`py-1 text-right ${pack.needed > 0 ? 'text-orange' : ''}`}>
                                            {pack.needed > 0 ? `x${pack.needed}` : '—'}
                                        </td>
                                        <td className="py-1 text-right">
                                            {lifetime ? formatNumber(pack.produced) : formatRate(pack.produced)}
                                        </td>
                                        <td className="py-1 text-right">
                                            {lifetime ? formatNumber(pack.consumed) : formatRate(pack.consumed)}
                                        </td>
                                    </tr>
                                ))}
                                </tbody>
                            </table>
                        </div>
                    </>
                }
            />
        </>
    );
};

export default Science;
