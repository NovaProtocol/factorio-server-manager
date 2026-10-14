import React, {useState} from "react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faCheck, faTimes} from "@fortawesome/free-solid-svg-icons";
import Panel from "../../../components/Panel";
import Button from "../../../components/Button";
import modsResource from "../../../../api/resources/mods";

const CompanionMod = ({companion, serverRunning, onToggle}) => {
    const [isToggling, setIsToggling] = useState(false);

    const installed = Boolean(companion);
    const enabled = installed && companion.enabled;
    const canToggle = installed && !serverRunning;

    const toggle = () => {
        setIsToggling(true);
        Promise.resolve(onToggle('fsm-companion'))
            .finally(() => setIsToggling(false));
    };

    const status = !installed
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
                            <div className="flex items-center">
                                <FontAwesomeIcon
                                    className={status.className}
                                    icon={enabled ? faCheck : faTimes}
                                />
                                <span className={`ml-2 font-bold ${status.className}`}>{status.text}</span>
                                {installed &&
                                    <span className="ml-2 text-gray-light">fsm-companion {companion.version}</span>
                                }
                            </div>
                            <p className="mt-2 max-w-3xl text-sm text-gray-light">
                                Some of this server manager's extra features are provided by the
                                FSM Companion mod, which the manager installs and keeps on your server.
                                With it installed and enabled those features work. Switch it off and they
                                are unavailable, and nothing else about your server changes.
                            </p>
                        </div>
                        <div className="ml-6 shrink-0 text-right">
                            <div className="flex items-center">
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
                            {!installed &&
                                <div className="mt-2 text-xs text-gray-light">
                                    The manager reinstalls it on its next start.
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