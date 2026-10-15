import Panel from "../components/Panel";
import React, {useEffect, useRef, useState} from "react";
import socket from "../../api/socket";

const Console = ({serverStatus}) => {

    const [logs, setLogs] = useState([]);
    const consoleInput = useRef(null);

    useEffect(() => {

        const appendLog = line => {
            setLogs(lines => [...lines, line]);
        }

        socket.on('gamelog', appendLog)
        socket.emit('log subscribe')
        consoleInput.current?.focus();

        return () => {
            socket.off('gamelog', appendLog);
            socket.emit("log unsubscribe")
        }
    }, []);

    return (
        <Panel
            title="Console"
            content={
                serverStatus.running
                    ? <>
                        <ul>
                            {logs?.map((log, i) => (<li key={i}>{log}</li>))}
                        </ul>
                        <input type="text"
                               className="shadow appearance-none border w-full py-2 px-3 text-black"
                               ref={consoleInput}
                               onKeyDown={e => {
                                   if (e.key === "Enter" && socket) {
                                       socket.emit("command send", consoleInput.current.value);
                                       consoleInput.current.value = ""
                                   }
                               }}
                        />
                        {/* Everything here is passed to the game as typed, so the one
                            way to hurt a save from this page is to type a script
                            command. The manager itself never sends one. */}
                        <p className="text-gray-light text-sm mt-3">
                            Commands run in the game exactly as typed. Script commands
                            (<code>/c</code>, <code>/sc</code>, <code>/silent-command</code>) permanently
                            disable achievements for the save, so the manager never sends one on
                            your behalf — the dashboards and the speed control use the companion
                            mod's own commands instead.
                        </p>
                    </>
                    : <p className="text-red-light pt-4">
                        The console is not available, because Factorio is not running.
                    </p>
            }
        />
    )
}

export default Console;
