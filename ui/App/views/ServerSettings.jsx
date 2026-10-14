import Panel from "../components/Panel";
import React, {useEffect, useState} from "react";
import settingsResource from "../../api/resources/settings";
import Button from "../components/Button";
import {CheckboxField, NumberField, SettingGroup, SettingRow, TextField} from "../components/SettingsControls";

// The same shape as the game settings page: a label, one control, and the game's
// own explanation behind a question mark. Booleans are checkboxes, numbers take
// numbers only, and a field still holding the value the game ships is marked
// "(default)".
const prettify = key => key.replaceAll('_', ' ');

// Grouped the way the server settings read, so related options sit together.
// Anything the game adds that is not listed here lands in "Other" rather than
// disappearing from the page.
const CATEGORIES = [
    {
        title: 'General',
        keys: ['name', 'description', 'tags', 'visibility', 'max_players',
               'ignore_player_limit_for_returning_players', 'require_user_verification'],
    },
    {
        title: 'Gameplay',
        keys: ['allow_commands', 'admins', 'auto_pause', 'auto_pause_when_players_connect',
               'only_admins_can_pause_the_game', 'afk_autokick_interval'],
    },
    {
        title: 'Networking',
        keys: ['max_upload_in_kilobytes_per_second', 'max_upload_slots',
               'minimum_latency_in_ticks', 'max_heartbeats_per_second'],
    },
    {
        title: 'Map streaming',
        keys: ['maximum_segment_size', 'maximum_segment_size_peer_count',
               'minimum_segment_size', 'minimum_segment_size_peer_count'],
    },
    {
        title: 'Autosave',
        keys: ['autosave_interval', 'autosave_slots', 'autosave_only_on_server',
               'non_blocking_saving'],
    },
    {
        title: 'Authentication',
        keys: ['username', 'password', 'token', 'game_password'],
    },
];

const ServerSettings = () => {

    const [settings, setSettings] = useState();
    const [defaults, setDefaults] = useState({});
    const [edits, setEdits] = useState({});
    const [isSaving, setIsSaving] = useState(false);

    const load = () => settingsResource.server.list().then(setSettings);

    useEffect(() => {
        load();
        settingsResource.server.defaults()
            .then(setDefaults)
            .catch(() => setDefaults({}));
    }, []);

    const current = key => (edits[key] !== undefined ? edits[key] : settings[key]);
    const setValue = (key, value) => setEdits(all => ({...all, [key]: value}));

    const noteFor = key =>
        defaults[key] !== undefined && String(defaults[key]) === String(current(key)) ? '(default)' : null;

    const save = async () => {
        setIsSaving(true);

        const payload = {...settings};
        Object.keys(edits).forEach(key => {
            payload[key] = typeof settings[key] === 'number' ? Number(edits[key]) : edits[key];
        });

        try {
            await settingsResource.server.update(payload);
            await load();
            setEdits({});
            window.flash('Server settings saved.', 'green');
        } finally {
            setIsSaving(false);
        }
    };

    const rows = [];

    if (settings) {
        Object.keys(settings).forEach(key => {
            if (key.startsWith('_comment_')) {
                return;
            }

            const value = current(key);
            const comment = settings['_comment_' + key];
            const description = Array.isArray(comment) ? comment.join(' ') : comment;

            if (key === 'visibility' && value && typeof value === 'object') {
                rows.push(
                    <SettingRow key={key} label={prettify(key)} description={description}>
                        <div className="flex flex-wrap items-center">
                            {Object.keys(value).map(scope => (
                                <label key={scope} className="mr-4 flex items-center">
                                    <CheckboxField checked={Boolean(value[scope])}
                                                   onChange={checked => setValue('visibility', {...value, [scope]: checked})}/>
                                    <span className="ml-2">{scope}</span>
                                </label>
                            ))}
                        </div>
                    </SettingRow>
                );
                return;
            }

            if (Array.isArray(value)) {
                rows.push(
                    <SettingRow key={key} label={prettify(key)} note={noteFor(key)} description={description}>
                        <TextField value={value.join(', ')}
                                   onChange={next => setValue(key, next.split(',').map(part => part.trim()).filter(Boolean))}/>
                    </SettingRow>
                );
                return;
            }

            let control;
            if (typeof value === 'boolean') {
                control = <CheckboxField checked={Boolean(value)}
                                        onChange={checked => setValue(key, checked)}/>;
            } else if (typeof value === 'number') {
                control = <NumberField value={String(value)}
                                       onChange={next => setValue(key, next)}/>;
            } else {
                control = <TextField value={value === null || value === undefined ? '' : String(value)}
                                     onChange={next => setValue(key, next)}/>;
            }

            rows.push(
                <SettingRow key={key} label={prettify(key)} note={noteFor(key)} description={description}>
                    {control}
                </SettingRow>
            );
        });
    }

    const grouped = new Map(CATEGORIES.map(category => [category.title, []]));
    grouped.set('Other', []);

    rows.forEach(row => {
        const category = CATEGORIES.find(entry => entry.keys.includes(row.key));
        grouped.get(category ? category.title : 'Other').push(row);
    });

    const content = (
        <>
            {Array.from(grouped.entries())
                .filter(([, groupRows]) => groupRows.length)
                .map(([title, groupRows]) => (
                    <React.Fragment key={title}>
                        <SettingGroup title={title}/>
                        {groupRows}
                    </React.Fragment>
                ))}
        </>
    );

    return (
        <form className="mb-4" onSubmit={event => { event.preventDefault(); save(); }}>
            <Panel
                title="Server Settings"
                content={content}
                actions={
                    <Button size="sm" type="success" isLoading={isSaving} onClick={save}>Save</Button>
                }
            />
        </form>
    )
}

export default ServerSettings;
