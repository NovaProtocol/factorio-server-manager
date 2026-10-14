import React, {useEffect, useState} from "react";
import savesResource from "../../../api/resources/saves";
import Panel from "../../components/Panel";
import CreateSaveForm from "./components/CreateSaveForm";
import UploadSaveForm from "./components/UploadSaveForm";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faDownload, faTrashAlt} from "@fortawesome/free-solid-svg-icons";

const Saves = ({serverStatus}) => {

    const [saves, setSaves] = useState([]);
    const [saveInfo, setSaveInfo] = useState({});

    const loadSaveInfo = (name) => {
        savesResource.info(name)
            .then(info => setSaveInfo(current => ({...current, [name]: info})))
            .catch(() => setSaveInfo(current => ({...current, [name]: null})));
    };

    const updateList = () => {
        savesResource.list()
            .then(res => {
                if (res) {
                    setSaves(res);
                    res.forEach(save => loadSaveInfo(save.name));
                }
            })

    }

    useEffect(() => {
        updateList();
    }, []);

    const deleteSave = async (save) => {
        const res = await savesResource.delete(save);
        if (res) {
            updateList()
        }
    }

    // What the save header itself says. The header's mod list is a snapshot from
    // creation time and drifts from what the world runs, so no mod count is shown
    // here; the build and level are stable facts about the file.
    const headerFacts = save => {
        const info = saveInfo[save.name];

        if (info === undefined) {
            return <span className="text-gray-light">reading header ...</span>;
        }
        if (info === null) {
            return <span className="text-gray-light">header unreadable</span>;
        }

        return (
            <>
                <span className="mr-3">level {info.name}</span>
                <span className="mr-3">build {info.loaded_from_build}</span>
                <span>difficulty {info.difficulty}</span>
            </>
        );
    };

    return (
        <>
            <div className="lg:flex mb-6">
                <Panel
                    title="Create Save"
                    className="lg:w-1/2 lg:mr-3 mb-6 lg:mb-0"
                    content={
                        serverStatus.running
                            ? <p className="text-red-light pt-4 pb-24">
                                Create a new Save is only possible if the Factorio server is
                                not running.
                            </p>
                            : <CreateSaveForm onSuccess={updateList}/>
                    }
                />
                <Panel
                    title="Upload Save"
                    className="lg:w-1/2 lg:ml-3"
                    content={<UploadSaveForm onSuccess={updateList}/>}
                />
            </div>

            <Panel
                className="mb-4"
                title="Saves"
                content={
                    <div className="overflow-x-auto w-full">
                        <table className="w-full">
                            <thead>
                            <tr className="text-left py-1">
                                <th>Name</th>
                                <th>Factorio Version</th>
                                <th>Last Modified At</th>
                                <th>Size</th>
                                <th>Actions</th>
                            </tr>
                            </thead>
                            <tbody>
                            {saves.map(save => {
                                const info = saveInfo[save.name];

                                return (
                                    <tr className="py-2 md:py-1" key={save.name}>
                                        <td className="pr-4">
                                            <div>{save.name}</div>
                                            <div className="text-sm text-gray-light">{headerFacts(save)}</div>
                                        </td>
                                        <td className="pr-4">
                                            {info ? info.factorio_version : <span className="text-gray-light">-</span>}
                                        </td>
                                        <td className="pr-4">{(new Date(save.last_mod)).toLocaleString()}</td>
                                        <td className="pr-4">{parseFloat(save.size / 1024 / 1024).toFixed(3)} MB</td>
                                        <td>
                                            <a href={`/api/saves/dl/${save.name}`} className="mr-2">
                                                <FontAwesomeIcon
                                                    className="text-gray-light cursor-pointer hover:text-orange"
                                                    icon={faDownload}/>
                                            </a>
                                            <FontAwesomeIcon className="text-red cursor-pointer hover:text-red-light mr-2"
                                                             onClick={() => deleteSave(save)} icon={faTrashAlt}/>
                                        </td>
                                    </tr>
                                );
                            })}
                            </tbody>
                        </table>
                    </div>
                }
            />
        </>
    )
}

export default Saves;
