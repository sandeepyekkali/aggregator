import React, { useState } from 'react';
import SnapTradeReactRaw from 'snaptrade-react';

const SnapTradeSDK = SnapTradeReactRaw as unknown as React.ElementType;

interface SnapTradeConnectProps {
    onSuccess: () => void; // Callback to trigger a UI refresh when the user finishes connecting
}

export const SnapTradeConnect: React.FC<SnapTradeConnectProps> = ({ onSuccess }) => {
    const [loginLink, setLoginLink] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // 1. Fetch the secure, user-specific portal URL from our Go backend
    const initializeConnection = async () => {
        setIsLoading(true);
        setError(null);
        
        try {
            const response = await fetch('/api/v1/snaptrade/link', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (!response.ok) {
                throw new Error('Failed to generate connection link');
            }

            const data = await response.json();
            setLoginLink(data.redirect_uri);
        } catch (err) {
            setError('Could not connect to brokerage gateway. Please try again.');
            console.error(err);
        } finally {
            setIsLoading(false);
        }
    };

    // 2. Render the trigger button if we haven't fetched the link yet
    if (!loginLink) {
        return (
            <div className="flex flex-col gap-2">
                <button
                    onClick={initializeConnection}
                    disabled={isLoading}
                    className="px-6 py-3 bg-blue-600 text-white rounded-md font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                >
                    {isLoading ? 'Preparing Secure Portal...' : 'Connect Premium Brokerage'}
                </button>
                {error && <p className="text-red-500 text-sm mt-1">{error}</p>}
            </div>
        );
    }

    // 3. Render the official SDK Portal Modal using the type-casted component
    return (
        <SnapTradeSDK
            loginLink={loginLink}
            onSuccess={(connectionId: string) => {
                console.log('Brokerage connected successfully!', connectionId);
                setLoginLink(null); 
                onSuccess(); 
            }}
            onError={(error: any) => {
                console.error('SnapTrade connection failed', error);
                setError('Connection failed or was interrupted.');
                setLoginLink(null);
            }}
            onExit={() => {
                setLoginLink(null); 
            }}
        />
    );
};