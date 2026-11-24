import {useEffect, useRef, useState} from 'react'
import DiscoverySettings,{discoveryDefaults,type Discovery} from './DiscoverySettings'
import SessionControls from './SessionControls'
import {ApiError, errorMessage, request} from './api'

import {avatars} from './avatars'

type Settings = {
  avatar?:string
  max_video_minutes?:number
  discovery:Discovery
  name: string
  navigation: string
  captions: boolean
  audio_description: boolean
  large_text:boolean
  high_contrast:boolean
  ending_music:boolean
  reduced_motion: boolean
}
type Profile = Settings & {id: string}
const defaults: Settings = {discovery:discoveryDefaults,name:'', navigation:'visual', captions:false, audio_description:false, reduced_motion:false,large_text:false,high_contrast:false,ending_music:false}
const navigationModes: Record<string,string> = {visual:'Pictures', 'visual-text':'Pictures and text', text:'Text'}
const toggles = [['captions','Captions'], ['audio_description','Audio description'], ['reduced_motion','Reduced motion'], ['large_text','Larger text'], ['high_contrast','High contrast'], ['ending_music','Session-ending music']] as const

export default function Profiles({onExpired,onProfiles}: {onExpired: () => void;onProfiles:(present:boolean)=>void}) {
  const [profiles,setProfiles] = useState<Profile[]>([])
  const [loading,setLoading] = useState(true)
  const [error,setError] = useState('')
  const [notice,setNotice] = useState('')
  const [busy,setBusy] = useState(false)
  const [reload,setReload] = useState(0)
  const [sessionProfile,setSessionProfile]=useState<Profile>()
  const [editing,setEditing] = useState<string>()
  const [settings,setSettings] = useState<Settings>(defaults)
  const nameInput = useRef<HTMLInputElement>(null)

  useEffect(() => {
    let active = true
    setLoading(true)
    setError('')
    request<{profiles:Profile[]}>('profiles').then(result => {
      if (active) setProfiles(result.profiles)
    }).catch(error => {
      if (!active) return
      if (error instanceof ApiError && error.status === 401) onExpired()
      else setError(errorMessage(error))
    }).finally(() => {if (active) setLoading(false)})
    return () => {active = false}
  }, [reload,onExpired])

  useEffect(()=>{
    if(!loading && !error)onProfiles(profiles.length>0)
  },[loading,error,profiles,onProfiles])
  function reset() {
    setEditing(undefined)
    setSettings(defaults)
  }
  function edit({id,...values}:Profile) {
    setEditing(id)
    setSettings({...values,discovery:{...discoveryDefaults,...values.discovery}})
    setNotice('')
    nameInput.current?.focus()
  }
  async function mutate(operation:() => Promise<void>) {
    setBusy(true)
    setError('')
    setNotice('')
    try {await operation()} catch(error) {
      if(error instanceof ApiError && error.status===401) onExpired()
      else setError(errorMessage(error))
    } finally {setBusy(false)}
  }
  function save(event:React.FormEvent) {
    event.preventDefault()
    void mutate(async () => {
      const p = await request<Profile>(editing ? 'profiles/'+encodeURIComponent(editing) : 'profiles', editing ? 'PUT' : 'POST', settings)
      setProfiles(previous => editing ? previous.map(old => old.id===editing ? p : old) : [...previous,p])
      reset()
      setNotice('Saved '+p.name+'.')
    })
  }
  function remove(p:Profile) {
    if (!window.confirm('Remove '+p.name+' and their profile settings?')) return
    void mutate(async () => {
      await request('profiles/'+encodeURIComponent(p.id),'DELETE')
      setProfiles(previous => previous.filter(old => old.id!==p.id))
      if (editing===p.id) reset()
      if(sessionProfile?.id===p.id)setSessionProfile(undefined)
      setNotice('Removed '+p.name+'.')
    })
  }
  return <>
    <h2>Child profiles</h2>
    {loading && <p role="status">Loading profiles…</p>}
    {error && <p role="alert">{error} <button onClick={() => setReload(n => n+1)}>Reload profiles</button></p>}
    {!error && !loading && !profiles.length && <p>Add a profile to choose a child’s viewing preferences.</p>}
    <ul>{profiles.map(p => <li key={p.id}>
      <strong><span aria-hidden="true">{avatars[p.avatar||'']}</span> {p.name}</strong> · {[navigationModes[p.navigation],...toggles.filter(([key])=>p[key]).map(([,label])=>label)].filter(Boolean).join(' · ')}
      <div className="actions">
        <button disabled={busy} aria-label={'Viewing session for '+p.name} onClick={()=>setSessionProfile(p)}>Session</button>
        <button disabled={busy} onClick={() => edit(p)} aria-label={'Edit '+p.name}>Edit</button>
        <button disabled={busy} onClick={() => remove(p)} aria-label={'Remove '+p.name}>Remove</button>
      </div>
    </li>)}</ul>
    {sessionProfile&&<SessionControls key={sessionProfile.id} profile={sessionProfile} close={()=>setSessionProfile(undefined)}/>}
    <form onSubmit={save}>
      <h3>{editing ? 'Edit profile' : 'New profile'}</h3>
      <label htmlFor="child-name">Profile name</label>
      <input ref={nameInput} id="child-name" value={settings.name} maxLength={50} required onChange={e => setSettings({...settings,name:e.target.value})}/>
      <label>Profile avatar<select value={settings.avatar||''} onChange={e=>setSettings({...settings,avatar:e.target.value})}>
        <option value="">None</option>{Object.entries(avatars).map(([name,symbol])=><option value={name} key={name}>{symbol} {name}</option>)}
      </select></label>
      <label>Maximum video length in minutes (0 for no limit)<input type="number" min="0" max="1440" required value={settings.max_video_minutes||0} onChange={e=>setSettings({...settings,max_video_minutes:Number(e.target.value)})}/></label>
      <label htmlFor="navigation">Navigation</label>
      <select id="navigation" value={settings.navigation} onChange={e => setSettings({...settings,navigation:e.target.value})}>
        {Object.entries(navigationModes).map(([value,label]) => <option key={value} value={value}>{label}</option>)}
      </select>
      <DiscoverySettings value={settings.discovery} change={discovery=>setSettings({...settings,discovery})}/>
      {toggles.map(([key,label]) => <label className="toggle" key={key}><input type="checkbox" checked={settings[key]} onChange={e=>setSettings({...settings,[key]:e.target.checked})}/> {label}</label>)}
      <button disabled={busy||loading}>{editing ? 'Save changes' : 'Add profile'}</button>
      {editing && <button type="button" onClick={reset} disabled={busy}>Cancel</button>}
    </form>
    <p role="status">{notice}</p>
    <a href="/api/profiles/export" download>Export profile settings</a>
  </>
}
