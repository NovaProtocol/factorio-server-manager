import React, {useCallback, useEffect, useState} from "react";
import {useForm} from "react-hook-form";
import Panel from "../components/Panel";
import Button from "../components/Button";
import Input from "../components/Input";
import InputPassword from "../components/InputPassword";
import Label from "../components/Label";
import Error from "../components/Error";
import TabControl from "../components/Tabs/TabControl";
import Tab from "../components/Tabs/Tab";
import credentials from "../../api/resources/credentials";

const TOKEN_PAGE = "https://factorio.com/profile";
const MOD_PAGE = "https://mods.factorio.com/mod/";

const PasswordForm = ({onResult}) => {
    const {register, handleSubmit, formState: {errors}} = useForm();
    const [isSigningIn, setIsSigningIn] = useState(false);

    const onSubmit = async data => {
        setIsSigningIn(true);
        try {
            const result = await credentials.login(data.username, data.password);
            window.flash("Signed in to the Factorio mod portal.", "green");
            onResult(result);
        } catch (e) {
            window.flash(e?.response?.data || "Sign in failed. Check the username and password.", "red");
        } finally {
            setIsSigningIn(false);
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="mb-4">
                <Label text="Factorio username" htmlFor="username"/>
                <Input register={register('username', {required: true})} placeholder="Username"/>
                <Error error={errors.username} message="Username is required"/>
            </div>
            <div className="mb-4">
                <Label text="Factorio password" htmlFor="password"/>
                <InputPassword register={register('password', {required: true})}/>
                <Error error={errors.password} message="Password is required"/>
            </div>
            <Button type="success" isLoading={isSigningIn} isSubmit={true}>Sign In</Button>
        </form>
    );
};

const TokenForm = ({onResult}) => {
    const {register, handleSubmit, formState: {errors}} = useForm();
    const [isSigningIn, setIsSigningIn] = useState(false);

    const onSubmit = async data => {
        setIsSigningIn(true);
        try {
            const result = await credentials.loginWithToken(data.username, data.token);
            window.flash("The token was accepted and stored.", "green");
            onResult(result);
        } catch (e) {
            window.flash(e?.response?.data || "The token was rejected.", "red");
        } finally {
            setIsSigningIn(false);
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="mb-4">
                <Label text="Factorio username" htmlFor="username"/>
                <Input register={register('username', {required: true})} placeholder="Username"/>
                <Error error={errors.username} message="Username is required"/>
            </div>
            <div className="mb-4">
                <Label text="Personal token" htmlFor="token"/>
                <Input register={register('token', {required: true})}
                       hasAutoComplete={false}
                       placeholder="Paste your token"/>
                <Error error={errors.token} message="Token is required"/>
            </div>
            <p className="mb-4 text-sm">
                <a className="text-blue hover:text-blue-light"
                   href={TOKEN_PAGE}
                   target="_blank"
                   rel="noreferrer">Where do I get my token?</a>
            </p>
            <Button type="success" isLoading={isSigningIn} isSubmit={true}>Sign In</Button>
        </form>
    );
};

const AccountData = ({status}) => {
    const bookmarks = status.bookmarks || [];
    const names = [...bookmarks].sort();

    return (
        <Panel
            title="Account data"
            className="mb-6"
            content={
                <>
                    <div className="mb-4">
                        <span className="font-bold text-dirty-white">Bookmarked mods: </span>
                        <span className="text-gray-light">{names.length}</span>
                    </div>
                    {names.length === 0
                        ? <p className="text-gray-light">This account has no bookmarked mods.</p>
                        : <ul className="flex flex-wrap gap-2">
                            {names.map(name => (
                                <li key={name}>
                                    <a className="accentuated bg-gray-light px-2 py-1 text-black hover:bg-orange"
                                       href={MOD_PAGE + name}
                                       target="_blank"
                                       rel="noreferrer">{name}</a>
                                </li>
                            ))}
                        </ul>
                    }
                </>
            }
        />
    );
};

const Credentials = () => {

    const [status, setStatus] = useState(null);
    const [isLoggingOut, setIsLoggingOut] = useState(false);
    const [isLoading, setIsLoading] = useState(true);

    const load = useCallback(async () => {
        try {
            setStatus(await credentials.status());
        } catch (e) {
            // the panel below reports the state; a failed read is not fatal
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        load();
    }, [load]);

    const logout = async () => {
        setIsLoggingOut(true);
        try {
            await credentials.logout();
            window.flash("Signed out of the Factorio mod portal.", "green");
            setStatus({loggedIn: false, username: "", bookmarks: [], error: ""});
        } catch (e) {
        } finally {
            setIsLoggingOut(false);
        }
    };

    const signedIn = Boolean(status?.loggedIn);

    return (
        <>
            <Panel
                title="Factorio Mod Portal"
                className="mb-6"
                content={
                    isLoading
                        ? <p className="text-gray-light">Checking the stored credentials…</p>
                        : signedIn
                            ? <>
                                <p className="mb-2">
                                    Signed in as <span className="font-bold text-dirty-white">{status.username}</span>.
                                </p>
                                {status.error &&
                                    <p className="mb-4 text-red-light">
                                        The mod portal rejected these credentials: {status.error}
                                    </p>
                                }
                                <Button type="danger" isLoading={isLoggingOut} onClick={logout}>Log Out</Button>
                            </>
                            : <TabControl>
                                <Tab title="Password">
                                    <PasswordForm onResult={setStatus}/>
                                </Tab>
                                <Tab title="Token">
                                    <TokenForm onResult={setStatus}/>
                                </Tab>
                            </TabControl>
                }
            />
            {signedIn && !status.error && <AccountData status={status}/>}
        </>
    );
};

export default Credentials;
