import React,{useCallback,useEffect,useState} from 'react'
import {createRoot} from 'react-dom/client'
import Viewer from './Viewer'
import Profiles from './Profiles'
import MediaLibrary from './MediaLibrary'
import {ApiError,errorMessage,request} from './api'
import './style.css'

type Phase='loading'|'setup'|'login'|'dashboard'
function ParentPin({onExpired}:{onExpired:()=>void}){
  const [pin,setPin]=useState('')
  const [busy,setBusy]=useState(false)
  const [message,setMessage]=useState('')
  async function save(event:React.FormEvent){
    event.preventDefault()
    setBusy(true);setMessage('')
    try{
      await request('admin/pin','PUT',{pin})
      setMessage(pin?'Parent PIN saved.':'Parent PIN removed.')
      setPin('')
    }catch(error){
      if(error instanceof ApiError && error.status===401)onExpired()
      else setMessage(errorMessage(error))
    }finally{setBusy(false)}
  }
  return <details><summary>Parent sign-in settings</summary><form onSubmit={save}>
    <p>Require a PIN along with the parent password. Saving signs out other parent browsers.</p>
    <label>New parent PIN (6–12 digits)<input type="password" inputMode="numeric" autoComplete="off" maxLength={12} pattern="[0-9]{6,12}" value={pin} onChange={e=>setPin(e.target.value)}/></label>
    <button disabled={busy}>{pin?'Save parent PIN':'Remove parent PIN'}</button>
    {message&&<p role="status">{message}</p>}
  </form></details>
}
function App() {
  const [phase,setPhase]=useState<Phase>('loading')
  const [pin,setPin]=useState('')
  const [hasProfiles,setHasProfiles]=useState<boolean>()
  const [hasApproval,setHasApproval]=useState<boolean>()
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
      await request(phase==='setup'?'setup':'login','POST',{password,pin})
      setPassword('');setPin('')
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
      {hasProfiles!==undefined && hasApproval!==undefined && (!hasProfiles || !hasApproval) && <nav aria-label="Household setup">
        <h2>Set up family viewing</h2>
        {hasProfiles ? <p><a href="#media-path">Add a video</a>, then choose Review to approve it for a child.</p> : <p><a href="#child-name">Create your first child profile</a> and choose how they browse.</p>}
        <p>Once a video is approved, open Session beside the child’s name to pair their screen.</p>
        <details><summary>Choosing what works for your family</summary>
          <p>Think about what your child enjoys, what they watch, and what helps them settle afterward. Decide what you want to leave time for and talk together about what you watched.</p>
          <a href="https://www.aap.org/en/patient-care/media-and-children/center-of-excellence-on-social-media-and-youth-mental-health/5-cs-quick-guide/">American Academy of Pediatrics guide to the 5 Cs</a>
        </details>
      </nav>}
      <Profiles onExpired={onExpired} onProfiles={setHasProfiles}/>
      <MediaLibrary onApproval={setHasApproval}/>
      <ParentPin onExpired={onExpired}/>
      {error&&<p role="alert">{error}</p>}
      <p><button onClick={logout}>Sign out</button></p>
    </section> : <section aria-labelledby="heading">
      <h1 id="heading">{phase==='setup'?'Create parent account':'Parent sign in'}</h1>
      <p>{phase==='setup'?'Choose a parent password of at least 12 characters.':'Enter your parent password.'}</p>
      <form onSubmit={submit}>
        <label htmlFor="password">Password</label>
        <input id="password" type="password" autoComplete={phase==='setup'?'new-password':'current-password'} minLength={12} maxLength={256} required value={password} onChange={e=>setPassword(e.target.value)}/>
        <label htmlFor="parent-pin">{phase==='setup'?'Parent PIN (optional, 6–12 digits)':'Parent PIN (if configured)'}</label>
        <input id="parent-pin" type="password" inputMode="numeric" autoComplete="off" pattern="[0-9]{6,12}" maxLength={12} value={pin} onChange={e=>setPin(e.target.value)}/>
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
