import { useState, useMemo } from 'react';
import { Eye, EyeOff, Loader2, AlertCircle, CheckCircle2, ShieldCheck, KeyRound } from 'lucide-react';
import { supabase } from '../utils/supabase';

interface AuthViewProps {
  isRecovering?: boolean;
  onRecovered?: () => void;
}

export function AuthView({ isRecovering = false, onRecovered }: AuthViewProps) {
  // State machine for the view: 'login' | 'signup' | 'forgot' | 'reset'
  const [mode, setMode] = useState<'login' | 'signup' | 'forgot'>(isRecovering ? 'forgot' : 'login');
  
  // If the App passes down isRecovering, we force the UI into the 'reset' state
  const currentMode = isRecovering ? 'reset' : mode;

  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  
  const [error, setError] = useState('');
  const [successMsg, setSuccessMsg] = useState('');

  const passwordStrength = useMemo(() => {
    let score = 0;
    if (password.length >= 8) score++;
    if (/[A-Z]/.test(password)) score++;
    if (/[0-9]/.test(password)) score++;
    if (/[^A-Za-z0-9]/.test(password)) score++;
    return score;
  }, [password]);

  const getStrengthColor = () => {
    if (passwordStrength <= 1) return 'bg-rose-500';
    if (passwordStrength === 2) return 'bg-amber-500';
    if (passwordStrength === 3) return 'bg-emerald-400';
    return 'bg-emerald-600';
  };

  const handleAuth = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccessMsg('');
    setLoading(true);

    try {
      if (currentMode === 'login') {
        const { error: authError } = await supabase.auth.signInWithPassword({ email, password });
        if (authError) throw authError;
      
      } else if (currentMode === 'signup') {
        if (password !== confirmPassword) throw new Error('Passwords do not match.');
        if (passwordStrength < 3) throw new Error('Please choose a stronger password.');
        if (!acceptedTerms) throw new Error('You must accept the Terms of Service.');

        const { error: authError, data } = await supabase.auth.signUp({ email, password });
        if (authError) throw authError;
        
        if (data.user) {
          if (!data.session) setSuccessMsg('Registration successful! Check your email to verify.');
          
          await fetch('http://localhost:8080/api/v1/users', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: data.user.id, email: data.user.email })
          });
        }
      
      } else if (currentMode === 'forgot') {
        const { error: resetError } = await supabase.auth.resetPasswordForEmail(email, {
          redirectTo: window.location.origin, // Returns them to your app
        });
        if (resetError) throw resetError;
        setSuccessMsg('Password reset instructions sent. Please check your email.');
      
      } else if (currentMode === 'reset') {
        if (password !== confirmPassword) throw new Error('Passwords do not match.');
        if (passwordStrength < 3) throw new Error('Please choose a stronger password.');

        const { error: updateError } = await supabase.auth.updateUser({ password });
        if (updateError) throw updateError;
        
        setSuccessMsg('Password successfully updated!');
        if (onRecovered) setTimeout(onRecovered, 1500); // Wait a moment so they see success message
      }

    } catch (err: any) {
      setError(err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-50 dark:bg-[#09090b] p-4">
      <div className="w-full max-w-md p-8 bg-white dark:bg-[#18181b] rounded-2xl border border-zinc-200 dark:border-white/10 shadow-xl">
        
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-indigo-100 dark:bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 mb-4">
            {currentMode === 'forgot' || currentMode === 'reset' ? <KeyRound size={28} /> : <ShieldCheck size={28} />}
          </div>
          <h2 className="text-2xl font-bold text-zinc-900 dark:text-white">
            {currentMode === 'login' ? 'Welcome Back' : 
             currentMode === 'signup' ? 'Create Account' : 
             currentMode === 'forgot' ? 'Reset Password' : 'Set New Password'}
          </h2>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-2">
            {currentMode === 'forgot' ? 'Enter your email to receive recovery instructions.' :
             currentMode === 'reset' ? 'Enter your new secure password below.' :
             'Bank-level security for your financial data.'}
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 rounded-xl flex items-start gap-3">
            <AlertCircle size={18} className="text-rose-600 dark:text-rose-400 mt-0.5 shrink-0" />
            <p className="text-sm text-rose-800 dark:text-rose-300">{error}</p>
          </div>
        )}
        
        {successMsg && (
          <div className="mb-6 p-4 bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 rounded-xl flex items-start gap-3">
            <CheckCircle2 size={18} className="text-emerald-600 dark:text-emerald-400 mt-0.5 shrink-0" />
            <p className="text-sm text-emerald-800 dark:text-emerald-300">{successMsg}</p>
          </div>
        )}

        <form onSubmit={handleAuth} className="space-y-5">
          {/* Email: Shown in Login, Signup, and Forgot */}
          {currentMode !== 'reset' && (
            <div>
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">Email Address</label>
              <input 
                type="email" 
                value={email} 
                onChange={e => setEmail(e.target.value)} 
                className="w-full px-4 py-2.5 rounded-xl border border-zinc-300 dark:border-white/10 bg-white dark:bg-zinc-900/50 text-zinc-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all outline-none" 
                placeholder="you@example.com"
                required 
              />
            </div>
          )}

          {/* Password: Shown in Login, Signup, and Reset */}
          {currentMode !== 'forgot' && (
            <div>
              <div className="flex justify-between items-center mb-1.5">
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  {currentMode === 'reset' ? 'New Password' : 'Password'}
                </label>
                {currentMode === 'login' && (
                  <button 
                    type="button" 
                    onClick={() => setMode('forgot')}
                    className="text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:underline focus:outline-none"
                  >
                    Forgot password?
                  </button>
                )}
              </div>
              <div className="relative">
                <input 
                  type={showPassword ? 'text' : 'password'}
                  value={password} 
                  onChange={e => setPassword(e.target.value)} 
                  className="w-full pl-4 pr-12 py-2.5 rounded-xl border border-zinc-300 dark:border-white/10 bg-white dark:bg-zinc-900/50 text-zinc-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all outline-none" 
                  placeholder="••••••••"
                  required 
                />
                <button 
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 p-1"
                >
                  {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                </button>
              </div>
              
              {(currentMode === 'signup' || currentMode === 'reset') && password.length > 0 && (
                <div className="mt-3 space-y-2">
                  <div className="h-1.5 w-full bg-zinc-200 dark:bg-zinc-800 rounded-full overflow-hidden flex gap-1">
                    {[1, 2, 3, 4].map((level) => (
                      <div 
                        key={level} 
                        className={`h-full flex-1 transition-colors duration-300 ${passwordStrength >= level ? getStrengthColor() : 'bg-transparent'}`}
                      />
                    ))}
                  </div>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400">
                    Must contain 8+ characters, a number, uppercase, and special character.
                  </p>
                </div>
              )}
            </div>
          )}

          {/* Confirm Password: Shown in Signup and Reset */}
          {(currentMode === 'signup' || currentMode === 'reset') && (
            <div className="animate-in fade-in slide-in-from-top-2 duration-300">
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1.5">Confirm Password</label>
              <input 
                type={showPassword ? 'text' : 'password'}
                value={confirmPassword} 
                onChange={e => setConfirmPassword(e.target.value)} 
                className="w-full px-4 py-2.5 rounded-xl border border-zinc-300 dark:border-white/10 bg-white dark:bg-zinc-900/50 text-zinc-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent transition-all outline-none" 
                placeholder="••••••••"
                required 
              />
            </div>
          )}

          {currentMode === 'signup' && (
            <div className="flex items-center gap-3 animate-in fade-in duration-300 pt-2">
              <input 
                type="checkbox" 
                id="terms"
                checked={acceptedTerms}
                onChange={e => setAcceptedTerms(e.target.checked)}
                className="w-4 h-4 rounded border-zinc-300 text-indigo-600 focus:ring-indigo-500 bg-white dark:bg-zinc-900 dark:border-zinc-700"
              />
              <label htmlFor="terms" className="text-sm text-zinc-600 dark:text-zinc-400">
                I agree to the <a href="#" className="text-indigo-600 dark:text-indigo-400 hover:underline">Terms of Service</a>
              </label>
            </div>
          )}

          <button 
            type="submit" 
            disabled={loading || !!successMsg}
            className="w-full py-2.5 px-4 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold rounded-xl transition-all disabled:opacity-70 flex justify-center items-center gap-2 mt-2"
          >
            {loading && <Loader2 size={18} className="animate-spin" />}
            {currentMode === 'login' ? 'Sign In' : 
             currentMode === 'signup' ? 'Create Account' : 
             currentMode === 'forgot' ? 'Send Reset Link' : 'Update Password'}
          </button>
        </form>
        
        {/* Footer Navigation Links */}
        {currentMode !== 'reset' && (
          <div className="mt-8 pt-6 border-t border-zinc-200 dark:border-white/10 text-center space-y-2">
            {currentMode === 'forgot' ? (
              <button 
                onClick={() => { setMode('login'); setError(''); setSuccessMsg(''); }} 
                className="text-sm font-semibold text-indigo-600 dark:text-indigo-400 hover:underline focus:outline-none"
              >
                Back to Sign In
              </button>
            ) : (
              <p className="text-sm text-zinc-600 dark:text-zinc-400">
                {currentMode === 'login' ? "Don't have an account?" : "Already have an account?"}
                <button 
                  onClick={() => { setMode(currentMode === 'login' ? 'signup' : 'login'); setError(''); setSuccessMsg(''); }} 
                  className="ml-2 font-semibold text-indigo-600 dark:text-indigo-400 hover:underline focus:outline-none"
                >
                  {currentMode === 'login' ? "Sign up" : "Sign in"}
                </button>
              </p>
            )}
          </div>
        )}
        
      </div>
    </div>
  );
}