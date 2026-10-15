import Panel from "../components/Panel";
import Button from "../components/Button";
import ConfirmDialog from "../components/ConfirmDialog";
import React, {useEffect, useState} from "react";
import settingsResource from "../../api/resources/settings";
import TabControl from "../components/Tabs/TabControl";
import Tab from "../components/Tabs/Tab";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faLock, faUndo} from "@fortawesome/free-solid-svg-icons";
import {
    CheckboxField,
    Dropdown,
    NumberField,
    SettingGroup,
    SettingRow,
    TextField,
} from "../components/SettingsControls";

// The three sections Factorio keeps in mod-settings.dat. The Player tab stays
// even though a dedicated server only applies per player settings to players
// that are actually connected.
const SECTIONS = [
    {
        key: 'startup',
        title: 'Startup',
        note: 'Loaded before the game data, so a change here is only picked up when the server next starts.',
    },
    {
        key: 'runtime-global',
        title: 'Map',
        note: 'Map settings, which a mod can change while the game is running.',
    },
    {
        key: 'runtime-per-user',
        title: 'Player',
        note: 'Per player settings. A dedicated server applies them to connected players.',
    },
];

const unwrap = entry => (entry && typeof entry === 'object' && 'value' in entry) ? entry.value : entry;

const formatValue = value => {
    if (value === null || value === undefined) {
        return '--';
    }
    return typeof value === 'object' ? JSON.stringify(value) : String(value);
};

// A setting's declared type and bounds come from the mod's settings.lua, since
// mod-settings.dat only holds the value. Absent means the mod that declared it is
// not installed, so nothing about the value can be checked.
const declarationOf = entry =>
    (entry && typeof entry === 'object' && entry.declaration) ? entry.declaration : null;

const hasDefault = entry => {
    const declaration = declarationOf(entry);
    return Boolean(declaration) && declaration.default_value !== undefined && declaration.default_value !== null;
};

// A field left blank means "use the declared default", so a value that already is
// the default starts blank and the number shows only as a placeholder.
const initialText = entry => {
    const value = unwrap(entry);

    if (hasDefault(entry) && String(declarationOf(entry).default_value) === String(value)) {
        return '';
    }

    return value === null || value === undefined ? '' : String(value);
};

const defaultPlaceholder = entry => {
    if (!hasDefault(entry)) {
        return declarationOf(entry) ? 'Default: not set' : 'Default: unknown';
    }
    return `Default: ${formatValue(declarationOf(entry).default_value)}`;
};

// The game reports a readable label for a setting where the mod gives one, so
// prefer that and keep the raw name underneath.
const labelOf = (entry, name) => {
    const declaration = declarationOf(entry);
    return declaration && declaration.label ? declaration.label : name;
};

const descriptionOf = entry => {
    const declaration = declarationOf(entry);
    return declaration && declaration.description ? declaration.description : '';
};

const describe = entry => {
    const declaration = declarationOf(entry);
    if (!declaration) {
        return 'No enabled mod declares this setting.';
    }

    const parts = [];
    if (Array.isArray(declaration.allowed_values) && declaration.allowed_values.length > 0) {
        parts.push(`one of: ${declaration.allowed_values.map(formatValue).join(', ')}`);
    }
    if (declaration.minimum_value !== undefined && declaration.minimum_value !== null) {
        parts.push(`min ${formatValue(declaration.minimum_value)}`);
    }
    if (declaration.maximum_value !== undefined && declaration.maximum_value !== null) {
        parts.push(`max ${formatValue(declaration.maximum_value)}`);
    }

    return parts.join(' · ');
};

const settingKey = (section, name) => `${section}:${name}`;

const EditorRow = ({section, name, entry, text, isUnlocked, isDisabled, onChange, onRevert, onLock}) => {
    const declaration = declarationOf(entry);
    const locked = !declaration && !isUnlocked;
    const disabled = isDisabled || locked;
    const placeholder = defaultPlaceholder(entry);
    const numeric = declaration && (declaration.type === 'int-setting' || declaration.type === 'double-setting');
    const allowed = declaration && Array.isArray(declaration.allowed_values) ? declaration.allowed_values : null;
    const stored = unwrap(entry);
    const isBool = Boolean(declaration && declaration.type === 'bool-setting');

    const isDefault = (candidate) => declaration
        && declaration.default_value !== undefined
        && declaration.default_value !== null
        && String(declaration.default_value) === String(candidate);

    let control;
    if (isBool) {
        control = (
            <CheckboxField
                checked={text === '' ? stored === true : text === 'true'}
                disabled={disabled}
                onChange={checked => onChange(String(checked))}
            />
        );
    } else if (allowed && allowed.length > 0) {
        // No blank entry: the default is annotated on the value it belongs to
        // rather than offered as a second one.
        control = (
            <Dropdown
                value={text === '' ? String(stored) : text}
                disabled={disabled}
                onChange={onChange}
                options={allowed.map(value => ({
                    value: String(value),
                    label: formatValue(value),
                    isDefault: isDefault(value),
                }))}
            />
        );
    } else if (numeric) {
        control = <NumberField value={text} placeholder={placeholder} disabled={disabled} onChange={onChange}/>;
    } else {
        control = <TextField value={text} placeholder={placeholder} disabled={disabled} onChange={onChange}/>;
    }

    // A checkbox has no blank state, so "at the default" is shown as an
    // annotation instead. Everything else carries its bounds there.
    const note = isBool
        ? (text === '' ? '(default)' : null)
        : describe(entry);

    return (
        <SettingRow
            label={labelOf(entry, name)}
            note={note}
            description={descriptionOf(entry)}
            actions={
                <>
                    {text !== '' &&
                        <FontAwesomeIcon
                            className="mr-3 text-gray-light cursor-pointer hover:text-orange"
                            title="Clear this to use the default"
                            icon={faUndo}
                            onClick={onRevert}
                        />
                    }
                    {!declaration &&
                        <FontAwesomeIcon
                            className={(isUnlocked ? 'text-orange' : 'text-red') + ' cursor-pointer hover:text-orange'}
                            title={isUnlocked ? 'Unlocked, editing at your own risk' : 'No enabled mod declares this setting'}
                            icon={faLock}
                            onClick={onLock}
                        />
                    }
                </>
            }
        >
            {control}
        </SettingRow>
    );
};

const SettingsTable = ({section, settings, texts, unlocked, isDisabled, onText, onRevert, onLock}) => {
    const names = settings ? Object.keys(settings).sort() : [];

    if (names.length === 0) {
        return <p className="text-gray-light">Nothing in this section.</p>;
    }

    // Group by the mod each setting came from, so a row's origin is visible.
    // A setting whose mod has no locale entry cannot be attributed.
    const groups = new Map();
    names.forEach(name => {
        const declaration = declarationOf(settings[name]);
        const mod = (declaration && declaration.mod) || 'not attributed to a mod';
        if (!groups.has(mod)) {
            groups.set(mod, []);
        }
        groups.get(mod).push(name);
    });

    return (
        <div>
            {Array.from(groups.entries()).map(([mod, groupNames]) => (
                <React.Fragment key={mod}>
                    <SettingGroup
                        title={mod}
                        note={`${groupNames.length} ${groupNames.length === 1 ? 'setting' : 'settings'}`}
                    />
                    {groupNames.map(name => {
                        const entry = settings[name];
                        const key = settingKey(section, name);
                        const text = texts[key] !== undefined ? texts[key] : initialText(entry);

                        return (
                            <EditorRow key={key}
                                       section={section}
                                       name={name}
                                       entry={entry}
                                       text={text}
                                       isUnlocked={Boolean(unlocked[key])}
                                       isDisabled={isDisabled}
                                       onChange={value => onText(key, value)}
                                       onRevert={() => onText(key, '')}
                                       onLock={() => onLock(section, name)}/>
                        );
                    })}
                </React.Fragment>
            ))}
        </div>
    );
};

const GameSettings = ({serverStatus}) => {

    const [config, setConfig] = useState();
    const [modSettings, setModSettings] = useState();
    const [texts, setTexts] = useState({});
    const [unlocked, setUnlocked] = useState({});
    const [pendingLock, setPendingLock] = useState(null);
    const [isSaving, setIsSaving] = useState(false);
    const [isDumping, setIsDumping] = useState(false);
    const [saveError, setSaveError] = useState(null);

    const load = () => {
        // A parse failure must not leave the panel stuck on "reading", so it gets
        // its own state rather than hanging on the pending one.
        settingsResource.game.modSettings()
            .then(data => {
                setModSettings(data);
                setTexts({});
                setUnlocked({});
            })
            .catch(() => setModSettings({error: true}));
    };

    useEffect(() => {
        settingsResource.game.list().then(setConfig);
        load();
    }, []);

    const isRunning = Boolean(serverStatus && serverStatus.running);
    const sections = modSettings ? modSettings.sections : null;
    const reference = modSettings ? modSettings.reference : null;

    // The game has to be stopped for this: it takes an exclusive lock on its
    // directory while it runs, and the server would be holding it.
    const dump = async () => {
        setIsDumping(true);
        setSaveError(null);

        try {
            await settingsResource.game.dumpModSettings();
            load();
        } catch (error) {
            setSaveError(error && error.response ? formatValue(error.response.data) : 'The dump did not complete.');
        } finally {
            setIsDumping(false);
        }
    };

    const collectChanges = () => {
        if (!sections) {
            return [];
        }

        const changes = [];

        SECTIONS.forEach(section => {
            const entries = sections[section.key];
            if (!entries) {
                return;
            }

            Object.keys(entries).sort().forEach(name => {
                const entry = entries[name];
                const key = settingKey(section.key, name);
                const text = texts[key];
                if (text === undefined || text === initialText(entry)) {
                    return;
                }

                // A blank field means the declared default. Where no default is
                // known there is nothing to write, so it is left alone.
                if (text === '' && !hasDefault(entry)) {
                    return;
                }

                changes.push({
                    section: section.key,
                    name,
                    value: text === '' ? String(declarationOf(entry).default_value) : text,
                    force: !declarationOf(entry),
                });
            });
        });

        return changes;
    };

    const save = async () => {
        const changes = collectChanges();
        if (changes.length === 0) {
            return;
        }

        setIsSaving(true);
        setSaveError(null);

        try {
            await settingsResource.game.saveModSettings(changes);
            load();
        } catch (error) {
            setSaveError(error && error.response ? formatValue(error.response.data) : 'Could not save the settings.');
        } finally {
            setIsSaving(false);
        }
    };

    const changeCount = () => collectChanges().length;

    return (
        <>
            <ConfirmDialog
                title="Setting not declared by any installed mod"
                confirmLabel="Unlock"
                isOpen={Boolean(pendingLock)}
                close={() => setPendingLock(null)}
                onSuccess={() => {
                    if (pendingLock) {
                        setUnlocked(current => ({...current, [settingKey(pendingLock.section, pendingLock.name)]: true}));
                    }
                    setPendingLock(null);
                    return Promise.resolve();
                }}
                content={
                    <>
                        <p className="mb-3">
                            <span className="text-dirty-white">{pendingLock ? pendingLock.name : ''}</span> does not
                            appear in any installed mod. The mod that declared it has probably been removed, so the
                            manager has no idea what type of value it takes.
                        </p>
                        <p className="mb-3">
                            Unlocking lets you write a value it cannot check. A wrong type can leave the settings
                            file broken and stop the server starting. A copy of the current file is kept beside it
                            as <span className="text-dirty-white">mod-settings.dat.bak</span> whenever the manager writes.
                        </p>
                        <p className="text-red font-bold">
                            Unlocking means you accept that risk.
                        </p>
                    </>
                }
            />

            <Panel
                className="mb-4"
                title="Game Settings"
                content={
                    !modSettings
                        ? <p className="text-gray-light">Reading mod-settings.dat ...</p>
                        : modSettings.error
                            ? <p className="text-red font-bold">
                                Could not read mod-settings.dat. The manager log has the parse error.
                            </p>
                            : modSettings.found
                            ? <>
                                {isRunning &&
                                    <p className="mb-3 text-red font-bold">
                                        The server is running. Stop it before changing these, since the game rewrites
                                        this file and would discard the change.
                                    </p>
                                }
                                {saveError &&
                                    <p className="mb-3 text-red font-bold">{saveError}</p>
                                }
                                {reference && !reference.present &&
                                    <p className="mb-3 text-orange font-bold">
                                        No settings reference yet, so nothing here can be checked against the
                                        mod that declares it. Every setting stays locked until you generate one.
                                    </p>
                                }
                                {reference && reference.present && !reference.current &&
                                    <p className="mb-3 text-orange font-bold">
                                        The settings reference was taken for a different set of enabled mods,
                                        so it is out of date. Generate it again to check these values against
                                        the mods you run now.
                                    </p>
                                }
                                <TabControl>
                                    {SECTIONS.map(section => (
                                        <Tab key={section.key} title={section.title}>
                                            <SettingsTable
                                                section={section.key}
                                                settings={sections ? sections[section.key] : null}
                                                texts={texts}
                                                unlocked={unlocked}
                                                isDisabled={isRunning}
                                                onText={(key, value) => setTexts(current => ({...current, [key]: value}))}
                                                onLock={(sectionKey, name) => setPendingLock({section: sectionKey, name})}
                                            />
                                        </Tab>
                                    ))}
                                </TabControl>
                            </>
                            : <div>
                                <p className="text-orange font-bold mb-2">No mod-settings.dat found.</p>
                                {isRunning &&
                                    <p className="mb-3 text-red font-bold">
                                        Stop the server first: the game needs exclusive use of its directory.
                                    </p>
                                }
                                {saveError &&
                                    <p className="mb-3 text-red font-bold">{saveError}</p>
                                }
                                <Button size="sm" type="success" isLoading={isDumping}
                                        isDisabled={isRunning} onClick={dump}>
                                    Pre-generate it
                                </Button>
                            </div>
                }
                actions={
                    modSettings && modSettings.found
                        ? <>
                            <Button size="sm" className="mr-2"
                                    isLoading={isDumping}
                                    isDisabled={isRunning}
                                    onClick={dump}>
                                {reference && reference.present ? 'Refresh settings reference' : 'Generate settings reference'}
                            </Button>
                            <Button size="sm" type="success"
                                    isLoading={isSaving}
                                    isDisabled={isRunning || changeCount() === 0}
                                    onClick={save}>
                                Save {changeCount() > 0 ? `${changeCount()} change${changeCount() === 1 ? '' : 's'}` : 'changes'}
                            </Button>
                        </>
                        : null
                }
            />

            <Panel
                className="mb-4"
                title="config.ini"
                content={
                    <>
                        {config && Object.keys(config).map(sectionKey => {
                            const settings = config[sectionKey];

                            return (
                                <div key={sectionKey}>
                                    <h1 className="mb-1 text-lg text-dirty-white">{sectionKey}</h1>
                                    <table className="w-full mb-2">
                                        <tbody>
                                        {settings && Object.keys(settings).length > 0
                                            ? Object.keys(settings).map(settingName => (
                                                <tr className="py-1" key={settingName}>
                                                    <td className="w-1/3 pr-4">{settingName}</td>
                                                    <td className="w-2/3 pr-4">{settings[settingName]}</td>
                                                </tr>
                                            ))
                                            : <tr>
                                                <td colSpan={2}>--</td>
                                            </tr>}
                                        </tbody>
                                    </table>
                                </div>
                            );
                        })}
                    </>
                }
            />
        </>
    )
}

export default GameSettings;
