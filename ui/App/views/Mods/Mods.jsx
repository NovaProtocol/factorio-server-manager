import Panel from "../../components/Panel";
import React, {useEffect, useState} from "react";
import modsResource from "../../../api/resources/mods";
import Button from "../../components/Button";
import server from "../../../api/resources/server";
import UploadMod from "./components/UploadMod";
import CreateModPack from "./components/CreateModPack";
import ModPack from "./components/ModPack";
import ModList from "./components/ModList";
import CompanionMod from "./components/CompanionMod";

const COMPANION_MOD_NAME = 'fsm-companion';

const Mods = ({serverStatus, currentUser}) => {

    // The `user` role may read the mod list but not change it. Mutating controls
    // are hidden and the API refuses them anyway.
    const readOnly = currentUser?.role !== 'admin';

    const [installedMods, setInstalledMods] = useState([]);
    const [modPacks, setModPacks] = useState([])
    const [factorioVersion, setFactorioVersion] = useState(null);
    const [isDeletingAllMods, setIsDeletingAllMods] = useState(false);
    const [isUpdatingAllMods, setIsUpdatingAllMods] = useState(false);
    const [updatableMods, setUpdatableMods] = useState([]);

    const addUpdatableMod = mod => {
        setUpdatableMods(mods => [...mods, mod])
    };

    const fetchInstalledMods = () => {
        modsResource.installed()
            .then(setInstalledMods);
    };

    const fetchModPacks = () => {
        modsResource.packs.list()
            .then(setModPacks)
    }

    const deleteAllMods = () => {
        setIsDeletingAllMods(true);
        modsResource.deleteAll()
            .then(fetchInstalledMods)
            .finally(() => setIsDeletingAllMods(false))
    }

    const updateAllMods = () => {
        setIsUpdatingAllMods(true);

        let promises = [];
        for (const updatableMod of updatableMods) {
            promises.push(modsResource.update(updatableMod))
        }

        Promise.all(promises)
            .then(fetchInstalledMods)
            .finally(() => setIsUpdatingAllMods(false));
    }

    useEffect(() => {
        server.factorioVersion()
            .then(data => {
                setFactorioVersion(data.base_mod_version)
                fetchInstalledMods();
                fetchModPacks();
            })

    }, []);

    const toggleMod = modName => {
        return modsResource
            .toggle(modName)
            .then(fetchInstalledMods)
    }

    const deleteMod = modName => {
        return modsResource
            .delete(modName)
            .then(fetchInstalledMods)
    }

    const updateMod = version => {
        return modsResource
            .update(version)
            .then(fetchInstalledMods)
    }

    let disabled = serverStatus.running
    // Reading is still allowed while the server runs or for a read-only account;
    // only the controls that change something are taken away.
    let controlsDisabled = disabled || readOnly

    // The companion is the manager's own mod: it gets its own panel instead of a
    // row in the list, so it is not offered for update or delete like a normal mod.
    const companionMod = installedMods.find(mod => mod.name === COMPANION_MOD_NAME);
    const listedMods = installedMods.filter(mod => mod.name !== COMPANION_MOD_NAME);

    return (
        <div>
            {disabled ?
                <Panel className="mb-6"
                       content={
                           <div className="text-red font-bold text-xl">
                               Changing mods is disabled while the server is running!
                           </div>
                       }
                />
                :
                !readOnly &&
                <Panel
                    title="Upload Mods"
                    className="mb-6"
                    content={<UploadMod refetchInstalledMods={fetchInstalledMods}/>}
                />
            }
            <Panel
                title="Mods"
                className="mb-6"
                content={
                    <ModList addUpdatableMod={addUpdatableMod}
                             toggleMod={toggleMod}
                             updateMod={updateMod}
                             deleteMod={deleteMod}
                             mods={listedMods}
                             factorioVersion={factorioVersion}
                             disabled={controlsDisabled}
                    />
                }
                actions={
                    <>
                        {
                            !controlsDisabled &&
                            <Button size="sm" className="mr-2" type="danger" isLoading={isDeletingAllMods}
                                    onClick={deleteAllMods}>Delete all Mods</Button> &&
                            <Button size="sm" className="mr-2" isLoading={isUpdatingAllMods}
                                    onClick={updateAllMods}>Update all Mods</Button>
                        }
                        <a className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold"
                           href={modsResource.downloadAllURL}>Download all Mods</a>
                    </>
                }
            />

            <CompanionMod
                companion={companionMod}
                serverRunning={disabled}
                readOnly={readOnly}
                onToggle={toggleMod}
                onInstalled={fetchInstalledMods}
            />

            <Panel
                title="Mod packs"
                className="mb-6"
                content={
                    modPacks.map(
                        (pack, i) =>
                            <ModPack factorioVersion={factorioVersion}
                                     key={i}
                                     modPack={pack}
                                     reloadMods={fetchInstalledMods}
                                     reloadModPacks={fetchModPacks}
                                     disabled={disabled}
                                     readOnly={readOnly}
                            />
                    )
                }
                actions={
                    !readOnly && <CreateModPack onSuccess={fetchModPacks}/>
                }
            />
        </div>
    )
}

export default Mods;
