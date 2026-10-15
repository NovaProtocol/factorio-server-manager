import React, {useMemo} from "react";
import {Line} from "react-chartjs-2";
import {
    CategoryScale,
    Chart as ChartJS,
    Filler,
    Legend,
    LinearScale,
    LineElement,
    PointElement,
    Tooltip,
} from "chart.js";

// Only the pieces this chart uses, so chart.js stays small in the bundle.
ChartJS.register(CategoryScale, LinearScale, LineElement, PointElement, Filler, Tooltip, Legend);

const ORANGE = '#E39827';
const BLUE = '#5C8FFF';
const GRID = '#403F40';
const MUTED = '#8E8E8E';

// The 300 samples behind a series, production and consumption overlaid as two
// lines the way the game's own statistics screen draws them. Hovering reads out
// both numbers for the same sample, which is the whole point of overlaying them.
const TrendChart = ({production, consumption, formatValue = String, height = 190, labels = {}}) => {

    const hasData = (production && production.length > 0) || (consumption && consumption.length > 0);

    const data = useMemo(() => {
        const length = Math.max(production?.length || 0, consumption?.length || 0);
        return {
            labels: Array.from({length}, (_, index) => index + 1),
            datasets: [
                {
                    label: labels.production || 'Produced',
                    data: production || [],
                    borderColor: ORANGE,
                    backgroundColor: ORANGE,
                    pointRadius: 0,
                    pointHitRadius: 8,
                    borderWidth: 2,
                },
                {
                    label: labels.consumption || 'Consumed',
                    data: consumption || [],
                    borderColor: BLUE,
                    backgroundColor: BLUE,
                    pointRadius: 0,
                    pointHitRadius: 8,
                    borderWidth: 2,
                },
            ],
        };
    }, [production, consumption, labels.production, labels.consumption]);

    const options = useMemo(() => ({
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        interaction: {mode: 'index', intersect: false},
        plugins: {
            legend: {
                display: true,
                labels: {color: '#ffe6c0', boxWidth: 12, boxHeight: 2, usePointStyle: false},
            },
            tooltip: {
                displayColors: true,
                backgroundColor: '#1C1C1C',
                borderColor: ORANGE,
                borderWidth: 1,
                titleColor: '#ffe6c0',
                bodyColor: '#F9F9F9',
                callbacks: {
                    title: items => `sample ${items[0].label} of ${data.labels.length} from the oldest`,
                    // Both series are listed by mode: 'index', which is the pair of
                    // numbers wanted here.
                    label: item => `${item.dataset.label}: ${formatValue(item.parsed.y)}`,
                },
            },
        },
        scales: {
            x: {display: false},
            y: {
                ticks: {color: MUTED, maxTicksLimit: 6, callback: value => formatValue(value)},
                grid: {color: GRID},
                border: {display: false},
            },
        },
    }), [formatValue, data.labels.length]);

    if (!hasData) {
        return <p className="text-gray-light">No graph data for this series yet.</p>;
    }

    return (
        <div style={{height}}>
            <Line data={data} options={options}/>
        </div>
    );
};

export default TrendChart;
