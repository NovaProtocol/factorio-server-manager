// Shared bits for the dashboard pages: the window list, the unit conversions and
// the number formatting.

// The eight windows Factorio's own statistics screen offers, plus all time. The
// tab labels are the short form ("1000h"), because nine of them have to fit on one
// row; the long form is the tooltip.
export const WINDOWS = [
    {value: '5s', name: '5s', title: '5 seconds'},
    {value: '1m', name: '1m', title: '1 minute'},
    {value: '10m', name: '10m', title: '10 minutes'},
    {value: '1h', name: '1h', title: '1 hour'},
    {value: '10h', name: '10h', title: '10 hours'},
    {value: '50h', name: '50h', title: '50 hours'},
    {value: '250h', name: '250h', title: '250 hours'},
    {value: '1000h', name: '1000h', title: '1000 hours'},
    {value: 'all', name: 'All', title: 'All time'},
];

export const windowName = value => (WINDOWS.find(w => w.value === value) || {}).name || value;

export const windowTitle = value => (WINDOWS.find(w => w.value === value) || {}).title || value;

// The game reports energy as joules per tick. There are 60 ticks in a second, so
// watts are simply 60x — this is the conversion that turns the raw API numbers
// into the kW/MW/GW the game itself shows.
export const TICKS_PER_SECOND = 60;

// "All time" on a power grid is a cumulative energy total, not a rate, so it has
// to be divided by the elapsed ticks to become a comparable average. Every other
// window is already an average of the same quantity.
export const reportWatts = (report, value) => {
    if (value === null || value === undefined || !report) {
        return null;
    }
    if (report.window === 'all') {
        return report.tick > 0 ? (value * TICKS_PER_SECOND) / report.tick : null;
    }
    return value * TICKS_PER_SECOND;
};

export const formatPower = watts => {
    if (watts === null || watts === undefined || Number.isNaN(watts)) {
        return '—';
    }

    const magnitude = Math.abs(watts);
    if (magnitude >= 1e9) return `${(watts / 1e9).toFixed(2)} GW`;
    if (magnitude >= 1e6) return `${(watts / 1e6).toFixed(1)} MW`;
    if (magnitude >= 1e3) return `${(watts / 1e3).toFixed(1)} kW`;
    return `${Math.round(watts)} W`;
};

export const formatEnergy = joules => {
    if (joules === null || joules === undefined) {
        return '—';
    }

    const magnitude = Math.abs(joules);
    if (magnitude >= 1e15) return `${(joules / 1e15).toFixed(2)} PJ`;
    if (magnitude >= 1e12) return `${(joules / 1e12).toFixed(2)} TJ`;
    if (magnitude >= 1e9) return `${(joules / 1e9).toFixed(2)} GJ`;
    if (magnitude >= 1e6) return `${(joules / 1e6).toFixed(1)} MJ`;
    if (magnitude >= 1e3) return `${(joules / 1e3).toFixed(1)} kJ`;
    return `${Math.round(joules)} J`;
};

export const formatNumber = value => (
    value === null || value === undefined ? '—' : Math.round(value).toLocaleString()
);

export const formatRate = value => (
    value === null || value === undefined ? '—' : `${Math.round(value).toLocaleString()}/min`
);

export const formatPercent = value => (
    value === null || value === undefined ? '—' : `${Math.round(value * 100)}%`
);
