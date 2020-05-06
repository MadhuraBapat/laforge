import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { CreateAccountComponent } from './create-account/create-account.component';
import { SignInAccountComponent } from './sign-in-account/sign-in-account.component';
import { CongratsOrganizerComponent } from './congrats-organizer/congrats-organizer.component';
import { CongratulationsComponent } from './congratulations/congratulations.component';
import { VolunteerComponent } from './volunteer/volunteer.component';

@NgModule({
  declarations: [
    AppComponent,
    CreateAccountComponent,
    SignInAccountComponent,
    CongratsOrganizerComponent,
    CongratulationsComponent,
    VolunteerComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule { }
