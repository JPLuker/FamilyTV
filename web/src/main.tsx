import React,{useCallback,useEffect,useState} from 'react'
import {createRoot} from 'react-dom/client'
import Viewer from './Viewer'
import Profiles from './Profiles'
import MediaLibrary from './MediaLibrary'
import {ApiError,errorMessage,request} from './api'
import './style.css'

type Phase='loading'|'setup'|'login'|'dashboard'
function App() {
  const [phase,setPhase]=useState<Phase>('loading')
  const [password,setPassword]=useState('')
  const [error,setError]=useState('')
  const [busy,setBusy]=useState(false)
  const [attempt,setAttempt]=useState(0)
  const onExpired=useCallback(()=>{setPhase('login');setError('Sign in again to continue.')},[])
  useEffect(()=>{
    let active=true
    async function connect(){
      try {
        const {needs_setup}=await request<{needs_setup:boolean}>('setup/status')
        if (!needs_setup) await request('admin')
        if (active) setPhase(needs_setup?'setup':'dashboard')
      } catch(error) {
        if (!active) return
        if(error instanceof ApiError && error.status===401) setPhase('login')
        else setError('Cannot connect to FamilyTV.')
      }
    }
    void connect()
    return ()=>{active=false}
  },[attempt])
  async function submit(e:React.FormEvent){
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await request(phase==='setup'?'setup':'login','POST',{password})
      setPassword('')
      setPhase(phase==='setup'?'login':'dashboard')
    } catch(error){setError(errorMessage(error))} finally {setBusy(false)}
  }
  async function logout(){
    setError('')
    try{await request('logout','POST');setPhase('login')}catch(error){setError(errorMessage(error))}
  }
  return <main>
    <header><strong>FamilyTV</strong><span>Parent dashboard · Development preview</span></header>
    {phase==='loading' ? <>
      <p role="status">{error||'Connecting…'}</p>
      {error && <button onClick={()=>{setError('');setAttempt(n=>n+1)}}>Retry connection</button>}
    </> : phase==='dashboard' ? <section aria-labelledby="welcome">
      <h1 id="welcome">Dashboard</h1>
      <Profiles onExpired={onExpired}/>
      <MediaLibrary/>
      {error&&<p role="alert">{error}</p>}
      <p><button onClick={logout}>Sign out</button></p>
    </section> : <section aria-labelledby="heading">
      <h1 id="heading">{phase==='setup'?'Create parent account':'Parent sign in'}</h1>
      <p>{phase==='setup'?'Choose a parent password of at least 12 characters.':'Enter your parent password.'}</p>
      <form onSubmit={submit}>
        <label htmlFor="password">Password</label>
        <input id="password" type="password" autoComplete={phase==='setup'?'new-password':'current-password'} minLength={12} maxLength={256} required value={password} onChange={e=>setPassword(e.target.value)}/>
        {error&&<p role="alert">{error}</p>}
        <button disabled={busy}>{busy?'Working…':phase==='setup'?'Create account':'Sign in'}</button>
      </form>
    </section>}
  </main>
}
createRoot(document.getElementById('root')!).render(<React.StrictMode>{location.pathname==='/watch'?<Viewer/>:<App/>}</React.StrictMode>)

if(location.pathname==='/' && 'serviceWorker' in navigator && window.isSecureContext){
  navigator.serviceWorker.register('/sw.js').catch(()=>{/* The dashboard also runs in a regular browser tab. */})
}
