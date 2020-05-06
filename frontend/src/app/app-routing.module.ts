import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

import { CreateAccountComponent } from './create-account/create-account.component';
import { SignInAccountComponent } from './sign-in-account/sign-in-account.component';
import { CongratulationsComponent } from './congratulations/congratulations.component';
import { CongratsOrganizerComponent } from './congrats-organizer/congrats-organizer.component';
import { VolunteerComponent } from './volunteer/volunteer.component';

const routes: Routes = [
  { path: '', component: CreateAccountComponent },
  { path: 'SignIn', component: SignInAccountComponent },
  { path: 'congratulations-volunteer', component: CongratulationsComponent },
  { path: 'congratulations-organizer', component: CongratsOrganizerComponent },
  { path: 'volunteer', component: VolunteerComponent }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
