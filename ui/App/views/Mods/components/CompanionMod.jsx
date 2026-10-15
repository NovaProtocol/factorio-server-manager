import React, {useCallback, useEffect, useState} from "react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faCheck, faTimes} from "@fortawesome/free-solid-svg-icons";
import Panel from "../../../components/Panel";
import Button from "../../../components/Button";
import modsResource from "../../../../api/resources/mods";

const SOURCE_LABEL = {
    github: 'GitHub release',
    portal: 'mod portal',
    installed: 'already installed',
};

const CompanionMod = ({companion, serverRunning, readOnly = false, onToggle, onInstalled}) => {

    const [status, setStatus] = useState(null);
    const [isToggling, setIsToggling] = useState(false);
    const [isFetching, setIsFetching] = useState(false);

    const loadStatus = useCallback(async () => {
        try {
            setStatus(await modsResource.companionStatus());
        } catch (e) {
            // the panel still renders from the installed mod list below
        }
    }, []);

    useEffect(() => {
        loadStatus();
    }, [loadStatus]);

    const installed = Boolean(companion);
    const enabled = installed && companion.enabled;
    const canToggle = installed && !serverRunning && !readOnly;

    const pinned = status?.pinned;
    const source = status?.source ? SOURCE_LABEL[status.source] : null;

    const toggle = () => {
        setIsToggling(true);
        Promise.resolve(onToggle('fsm-companion'))
            .finally(() => setIsToggling(false));
    };

    // Fetching is the manual retry: the manager also does this on every start.
    // A failure leaves the reason in the status, which is what the page shows.
    const fetchNow = () => {
        setIsFetching(true);
        modsResource.companionInstall()
            .then(setStatus)
            .catch(() => loadStatus())
            .finally(() => {
                setIsFetching(false);
                onInstalled?.();
            });
    };

    const state = !installed
        ? {text: 'Not installed', className: 'text-red'}
        : enabled
            ? {text: 'Installed and enabled', className: 'text-green'}
            : {text: 'Installed but disabled', className: 'text-orange'}

    return (
        <Panel
            title="FSM Companion"
            className="mb-6"
            content={
                <div>
                    <div className="flex items-center justify-between">
                        <div>
                            <div className="flex flex-wrap items-center">
                                <FontAwesomeIcon
                                    className={state.className}
                                    icon={enabled ? faCheck : faTimes}
                                />
                                <span className={`ml-2 whitespace-nowrap font-bold ${state.className}`}>{state.text}</span>
                                {installed &&
                                    <span className="ml-2 whitespace-nowrap text-gray-light">fsm-companion {companion.version}</span>
                                }
                                {pinned &&
                                    <span className="ml-2 whitespace-nowrap text-gray-light">(manager pins {pinned})</span>
                                }
                                {source &&
                                    <span className="ml-2 whitespace-nowrap text-gray-light">from the {source}</span>
                                }
                            </div>
                            <p className="mt-2 max-w-3xl text-sm text-gray-light">
                                Some of this server manager's extra features are provided by the
                                FSM Companion mod. The manager fetches the version it pins and keeps it
                                on your server; switch it off and those features are unavailable, and
                                nothing else about your server changes.
                            </p>
                            {status?.error &&
                                <p className="mt-2 max-w-3xl text-sm text-red-light">
                                    Could not fetch it: {status.error}
                                </p>
                            }
                        </div>
                        <div className="ml-6 shrink-0 text-right">
                            <div className="flex items-center">
                                {!readOnly && !installed &&
                                    <Button
                                        size="sm"
                                        className="mr-2"
                                        type="success"
                                        isLoading={isFetching}
                                        isDisabled={serverRunning}
                                        onClick={fetchNow}
                                    >
                                        Fetch now
                                    </Button>
                                }
                                {!readOnly && installed &&
                                    <Button
                                        size="sm"
                                        className="mr-2"
                                        type={enabled ? 'danger' : 'success'}
                                        isDisabled={!canToggle}
                                        isLoading={isToggling}
                                        onClick={toggle}
                                    >
                                        {enabled ? 'Disable' : 'Enable'}
                                    </Button>
                                }
                                <a className="mr-2 bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold"
                                   href={modsResource.companionDownloadURL}>Download</a>
                                <a className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold"
                                   href={modsResource.companionPortalURL}
                                   target="_blank"
                                   rel="noreferrer">Mod portal</a>
                            </div>
                            {serverRunning &&
                                <div className="mt-2 text-xs text-gray-light">
                                    Stop the server to change this.
                                </div>
                            }
                        </div>
                    </div>
                </div>
            }
        />
    )
}

export default CompanionMod;
