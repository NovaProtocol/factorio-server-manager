import React, {useEffect, useState} from "react";
import Panel from "../components/Panel";
import Button from "../components/Button";
import server from "../../api/resources/server";
import savesResource from "../../api/resources/saves";
import companion from "../../api/resources/companion";
import {useForm} from "react-hook-form";
import Select from "../components/Select";
import Input from "../components/Input";
import Error from "../components/Error";

const Controls = ({serverStatus}) => {

    const factorioVersion = serverStatus.fac_version ? serverStatus.fac_version : 'Unknown';
    const [saves, setSaves] = useState([]);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isStopping, setIsStopping] = useState(false);
    const [isStarting, setIsStarting] = useState(false);
    const [isKilling, setIsKilling] = useState(false);

    const { handleSubmit, reset, register, formState: {errors} } = useForm();

    const startServer = async (data) => {
        setIsStarting(true);
        await server.start(data.ip, parseInt(data.port), data.save);
    }

    const stopServer = async () => {
        setIsStopping(true);
        await server.stop();
    }

    const killServer = async () => {
        setIsKilling(true);
        await server.kill();
    }

    // Game speed and max UPS are one equation apart (max_ups = speed x 60), but
    // they are two editable fields, not one value rendered twice. While a field
    // has focus it owns what is in it: nothing recalculates or overwrites it, so
    // backspacing it empty stays empty instead of snapping back to a number. The
    // twin field is solved once, on focus-out, from whatever was typed there.
    const liveSpeed = serverStatus.game_speed ?? 1;
    const [speedText, setSpeedText] = useState(null);
    const [upsText, setUpsText] = useState(null);
    const [lastEdited, setLastEdited] = useState('speed');
    const [isSavingSpeed, setIsSavingSpeed] = useState(false);
    const [speedMessage, setSpeedMessage] = useState(null);

    // Six decimals is enough for a speed and short enough to read; 1 UPS comes
    // out as 0.016667 rather than 0.016666666666666666.
    const format = value => {
        if (value === null || value === undefined || isNaN(value)) {
            return '';
        }
        return String(Math.round(value * 1000000) / 1000000);
    };

    const parse = text => {
        const value = parseFloat(text);
        return isNaN(value) ? null : value;
    };

    const touched = speedText !== null || upsText !== null;
    const shownSpeed = speedText !== null ? speedText : format(liveSpeed);
    const shownUps = upsText !== null ? upsText : format(liveSpeed * 60);

    // What Save will send: the value as typed, not as displayed, so typing 1 UPS
    // sends exactly 1/60 of a speed instead of 0.016667 x 60.
    const pendingSpeed = () => {
        if (lastEdited === 'ups') {
            const ups = parse(shownUps);
            return ups === null ? null : ups / 60;
        }
        return parse(shownSpeed);
    };

    const editSpeed = text => {
        setLastEdited('speed');
        setSpeedText(text);
    };

    const editUps = text => {
        setLastEdited('ups');
        setUpsText(text);
    };

    // On focus-out the twin is solved. An empty or out-of-range field is dropped
    // rather than guessed at, so Save simply falls back to the value in play.
    const solveSpeed = () => {
        const speed = parse(shownSpeed);
        if (speed === null || speed < 0.01 || speed > 3) {
            setSpeedText(null);
            return;
        }
        setSpeedText(format(speed));
        setUpsText(format(speed * 60));
    };

    const solveUps = () => {
        const ups = parse(shownUps);
        if (ups === null || ups < 0.6 || ups > 180) {
            setUpsText(null);
            return;
        }
        setSpeedText(format(ups / 60));
        setUpsText(format(ups));
    };

    const saveSpeed = async () => {
        setIsSavingSpeed(true);
        setSpeedMessage(null);
        try {
            await companion.setSpeed(pendingSpeed());
            setSpeedText(null);
            setUpsText(null);
        } catch (e) {
            setSpeedMessage(e?.response?.data?.error ?? 'could not set the speed');
        }
        setIsSavingSpeed(false);
    };

    useEffect(() => {
        savesResource.list(true)
            .then(res => {
                setSaves(res);
                if (res.length > 0) {
                    setIsDisabled(undefined);
                }
                reset();
            });
    }, [])

    const speedPanel = running => {
        const runningState = running === undefined ? serverStatus.running : running;
        return (
            <Panel
                title="Game Speed"
                className="mt-4"
                content={
                    <div className="lg:flex">
                        <div className="lg:w-1/3 mb-2 mr-0 lg:mr-4">
                            <div className="font-bold">Game speed</div>
                            <Input
                                type="number"
                                step={0.01}
                                min={0.01}
                                max={3}
                                value={shownSpeed}
                                disabled={runningState ? undefined : true}
                                onChange={e => editSpeed(e.target.value)}
                                onBlur={solveSpeed}
                            />
                        </div>
                        <div className="lg:w-1/3 mb-2 mr-0 lg:mr-4">
                            <div className="font-bold">Max UPS</div>
                            <Input
                                type="number"
                                step={0.6}
                                min={0.6}
                                max={180}
                                value={shownUps}
                                disabled={runningState ? undefined : true}
                                onChange={e => editUps(e.target.value)}
                                onBlur={solveUps}
                            />
                        </div>
                        <div className="flex-1 mb-2">
                            <div className="font-bold">Running now</div>
                            <div>
                                {runningState
                                    ? <span>{(serverStatus.ups ?? 0).toFixed(1)} UPS at speed {format(liveSpeed)}</span>
                                    : <span className="text-gray-light">set the speed after the game starts</span>}
                            </div>
                        </div>
                    </div>
                }
                actions={
                    <div className="md:flex items-center">
                        {speedMessage ? <span className="text-red-light mr-3">{speedMessage}</span> : null}
                        {touched && runningState
                            ? <Button onClick={() => { setSpeedText(null); setUpsText(null); }} size="sm" type="default" className="w-full md:w-auto mb-2 md:mb-0 md:mr-2">Revert</Button>
                            : null}
                        <Button
                            onClick={saveSpeed}
                            isDisabled={runningState
                                ? (pendingSpeed() === null || pendingSpeed() < 0.01 || pendingSpeed() > 3 || pendingSpeed() === liveSpeed)
                                : true}
                            isLoading={isSavingSpeed}
                            size="sm"
                            type="success"
                            className="w-full md:w-auto">Save</Button>
                    </div>
                }
            />
        );
    };

    return (
        <form onSubmit={handleSubmit(startServer)}>
        <Panel
            title="Server Status"
            content={
                <div className="lg:flex">
                    { serverStatus.running
                        ? <>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Status</div>
                                <div>{serverStatus.running ? 'Running' : 'Stopped'}{serverStatus.ups !== undefined ? ` - ${serverStatus.ups.toFixed(1)} UPS` : ''}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">IP</div>
                                <div>{serverStatus.bindip}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Port</div>
                                <div>{serverStatus.port}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Factorio Version</div>
                                <div>{factorioVersion}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Save</div>
                                <div>{serverStatus.savefile}</div>
                            </div>
                        </>
                        : <>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Status</div>
                                <div>{serverStatus.running ? 'Running' : 'Stopped'}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">IP</div>
                                <Input
                                    defaultValue={"0.0.0.0"}
                                    disabled={isDisabled}
                                    register={register('ip',{required: true, pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/})}
                                />
                                <Error error={errors.ip} message="IP is required and must be valid."/>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">Port</div>
                                <Input
                                    type="number"
                                    min={1}
                                    max={65535}
                                    defaultValue={"34197"}
                                    disabled={isDisabled}
                                    register={register('port',{required: true, min: 1, max: 65535})}
                                />
                                <Error error={errors.port} message="Port is required within range 1-65535"/>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">Factorio Version</div>
                                <div>{factorioVersion}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Save</div>
                                <div className="relative">
                                    <Select
                                        register={register('save',{required: true})}
                                        defaultValue={saves.find((save) => save.name.startsWith('Load Latest'))?.name}
                                        disabled={isDisabled}
                                        options={saves.map(save => new Object({
                                            value: save.name,
                                            name: save.name
                                        }))}
                                    />
                                    <Error error={errors.save} message="Save is required and must be valid."/>
                                </div>
                            </div>
                        </>
                    }
                </div>
            }
            actions={
                <div className="md:flex">
                    {serverStatus.running
                        ? <>
                            <Button onClick={stopServer} isLoading={isStopping} isDisabled={isKilling} size="sm" className="w-full md:w-auto mb-2 md:mb-0 md:mr-2" type="default">Save & Stop Server</Button>
                            <Button onClick={killServer} isLoading={isKilling} isDisabled={isStopping} size="sm" type="danger" className="w-full md:w-auto">Kill Server</Button>
                        </>
                        : <Button isSubmit={true} isDisabled={isDisabled} isLoading={isStarting} size="sm" type="success" className="w-full md:w-auto">Start Server</Button>
                    }
                </div>
            }
        />
        {speedPanel()}
        </form>
    )
};

export default Controls;
